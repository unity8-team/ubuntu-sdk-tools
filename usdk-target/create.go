/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * Author: Benjamin Zeller <benjamin.zeller@canonical.com>
 */

package main

import (
	"fmt"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/gnuflag"
	"launchpad.net/ubuntu-sdk-tools"
	"os"
	"strings"
	"path/filepath"
)

type createCmd struct {
	architecture    string
	framework       string
	fingerprint     string
	name            string
	createSupGroups bool
}

func (c *createCmd) usage() string {
	return `\
Creates a new Ubuntu SDK build target.

usdk-target create -a ARCHITECTURE -f FRAMEWORK -n NAME -p FINGERPRINT
`
}

var requiredString = "REQUIRED"

func (c *createCmd) flags() {
	gnuflag.StringVar(&c.architecture, "a", requiredString, "architecture for the chroot")
	gnuflag.StringVar(&c.framework, "f", requiredString, "framework for the chroot")
	gnuflag.StringVar(&c.fingerprint, "p", requiredString, "sha256 fingerprint of the base image")
	gnuflag.StringVar(&c.name, "n", requiredString, "name of the container")
	gnuflag.BoolVar(&c.createSupGroups, "g", false, "Also try to create the users supplementary groups")
}



func (c *createCmd) run(args []string) error {
	if c.architecture == requiredString || c.framework == requiredString || c.fingerprint == requiredString || c.name == requiredString {
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments")
	}

	if os.Getuid() != 0 {
		return fmt.Errorf("This command needs to run as root")
	}

	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the LXD server.\n")
		os.Exit(1)
	}

	//name string, imgremote string, image string, profiles *[]string, config map[string]string, ephem bool
	var prof *[]string
	conf := make(map[string]string)

	conf["security.privileged"] = "true"
	conf["user.click-architecture"] = c.architecture
	conf["user.click-framework"] = c.framework
	conf["raw.lxc"] = "lxc.aa_profile = unconfined"

	resp, err := client.Init(c.name, "ubuntu-sdk-images", c.fingerprint, prof, conf, false)
	if err != nil {
		return err
	}

	c.initProgressTracker(client, resp.Operation)
	err = client.WaitForSuccess(resp.Operation)

	if err != nil {
		return err
	} else {
		op, err := resp.MetadataAsOperation()
		if err != nil {
			return fmt.Errorf("didn't get any affected image, container or snapshot from server")
		}

		containers, ok := op.Resources["containers"]
		if !ok || len(containers) == 0 {
			return fmt.Errorf("didn't get any affected image, container or snapshot from server")
		}

		if len(containers) == 1 && c.name == "" {
			fields := strings.Split(containers[0], "/")
			fmt.Printf("Container name is: %s\n", fields[len(fields)-1])
		}
	}

	//make the rootfs readable
	rootfs := shared.VarPath("containers", c.name)

	fi, err := os.Lstat(rootfs)
	if err != nil {
		ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
		return fmt.Errorf("Failed to make rootfs readable. error: %v.\n",err)
	}

	if fi.Mode() & os.ModeSymlink == os.ModeSymlink {
		rootfs, err = filepath.EvalSymlinks(rootfs)
		if err != nil {
			ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
			return fmt.Errorf("Failed to make rootfs readable. error: %v.\n",err)
		}
	}

	err = os.Chmod(rootfs, 0755)
	if err != nil {
		ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
		return fmt.Errorf("Failed to make rootfs readable. error: %v.\n",err)
	}

	//add the required devices
	err = ubuntu_sdk_tools.AddDeviceSync(client, c.name, "dri", "disk", []string{"source=/dev/dri", "path=/dev/dri"})
	if err != nil {
		ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
		return err
	}

	err = ubuntu_sdk_tools.AddDeviceSync(client, c.name, "tmp", "disk", []string{"source=/tmp", "path=/tmp"})
	if err != nil {
		ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
		return err
	}

	err = RegisterUserInContainer(client, c.name, nil, c.createSupGroups)
	if err != nil {
		ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
		return err
	}

	return nil
}

func (c *createCmd) initProgressTracker(d *lxd.Client, operation string) {
	handler := func(msg interface{}) {
		if msg == nil {
			return
		}

		event := msg.(map[string]interface{})
		if event["type"].(string) != "operation" {
			return
		}

		if event["metadata"] == nil {
			return
		}

		md := event["metadata"].(map[string]interface{})
		if !strings.HasSuffix(operation, md["id"].(string)) {
			return
		}

		if md["metadata"] == nil {
			return
		}

		if shared.StatusCode(md["status_code"].(float64)).IsFinal() {
			return
		}

		opMd := md["metadata"].(map[string]interface{})
		_, ok := opMd["download_progress"]
		if ok {
			fmt.Printf("Retrieving image: %s\n", opMd["download_progress"].(string))
		}

		if opMd["download_progress"].(string) == "100%" {
			fmt.Printf("\n")
		}
	}
	go d.Monitor([]string{"operation"}, handler)
}
