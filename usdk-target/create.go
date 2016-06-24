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
 *
 * Based on the LXD lxc client. Copyright Holders:
 * Author: Michael McCracken
 * Author: René Jochum
 * Author: Serge Hallyn
 * Author: Stéphane Graber
 * Author: Tycho Andersen
 * Author: benaryorg
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
	"regexp"
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

usdk-target create -n NAME -p FINGERPRINT
`
}

var requiredString = "REQUIRED"
var baseFWRegexNoMinor = regexp.MustCompile("^(ubuntu-[^-]+-[\\d]{1,2}\\.[\\d]{1,2})-([^-]+)-([^-]+)-([^-]+)?$")
var baseFWRegexWithMinor = regexp.MustCompile("^(ubuntu-[^-]+-[\\d]{1,2}\\.[\\d]{1,2})(\\.[\\d]+)-([^-]+)-([^-]+)-([^-]+)?$")

func (c *createCmd) flags() {
	gnuflag.StringVar(&c.architecture, "a", "", "architecture for the chroot (deprecated)")
	gnuflag.StringVar(&c.framework, "f", "", "framework for the chroot  (deprecated)")
	gnuflag.StringVar(&c.fingerprint, "p", requiredString, "sha256 fingerprint of the base image")
	gnuflag.StringVar(&c.name, "n", requiredString, "name of the container")
	gnuflag.BoolVar(&c.createSupGroups, "g", false, "Also try to create the users supplementary groups")
}



func (c *createCmd) run(args []string) error {
	if c.fingerprint == requiredString || c.name == requiredString {
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments")
	}

	if os.Getuid() != 0 {
		return fmt.Errorf("This command needs to run as root")
	}

	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, "ubuntu-sdk-images")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the remote LXD server.\n")
		os.Exit(1)
	}

	//get image informations
	images, err := findRelevantImages(client)
	if err != nil {
		return err
	}

	var requestedImage *imageDesc = nil
	for _, image := range images {
		if image.Fingerprint == c.fingerprint {
			requestedImage = &image
			break
		}
	}
	if requestedImage == nil {
		return fmt.Errorf("Could not find the requested image fingerprint: %s", c.fingerprint)
	}

	parts := baseFWRegexNoMinor.FindStringSubmatch(requestedImage.Alias)
	if len(parts) != 0 {
		c.framework = parts[1]
		c.architecture = parts[3]
	} else {
		parts := baseFWRegexWithMinor.FindStringSubmatch(requestedImage.Alias)
		if len(parts) == 0 {
			return fmt.Errorf("Alias format of image is unsupported: %s\n", requestedImage.Alias)
		}

		c.framework = parts[1]
		c.architecture = parts[4]
	}


	fmt.Printf("Creating image with:\nframework: %s\narch: %s\n", c.framework, c.architecture)
	client, err = lxd.NewClient(config, config.DefaultRemote)
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

	for _, fixable := range fixable_set {
		err = fixable.Fix(client)
		if err != nil {
			ubuntu_sdk_tools.RemoveContainerSync(client, c.name)
			return err
		}
	}

	//add the required devices
	err = ubuntu_sdk_tools.AddDeviceSync(client, c.name, "tmp", "disk", []string{"source=/tmp", "path=/tmp", "recursive=true"})
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
