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
package fixables

import (
	"fmt"
	"path/filepath"
	"github.com/lxc/lxd"
	"launchpad.net/ubuntu-sdk-tools"
	"os"
	"io/ioutil"
	"regexp"
	"github.com/lxc/lxd/shared"
	"strings"
)

var driverVerFile    = "/sys/module/nvidia/version"
var driverDirName    = "nv-bin"
var versionPattern   = regexp.MustCompile("^([0-9]+).*$")

var globNvDir *string = nil

type NvidiaFixable struct { }

func (*NvidiaFixable) findNvidiaDir ( ) (*string, error) {
	if (globNvDir != nil) {
		return globNvDir, nil
	}

	verBytes, err := ioutil.ReadFile(driverVerFile)
	if err != nil {
		return nil, err
	}

	verString := strings.TrimSpace(string(verBytes))
	parts := versionPattern.FindStringSubmatch(verString)
	if len(parts) == 0 {
		return nil, nil
	}

	nvidiaDir := fmt.Sprintf("/usr/lib/nvidia-%s", parts[1])
	if _, err := os.Stat(nvidiaDir); os.IsNotExist(err) {
		fmt.Printf("Nvidia dir %s does not exist\n", nvidiaDir)
		return nil, nil
	}

	globNvDir = &nvidiaDir
	return globNvDir, nil
}

func (c *NvidiaFixable) run(client *lxd.Client, container *shared.ContainerInfo, doFix bool) error {
	//we have no nvidia module loaded if this file does not exist
	if _, err := os.Stat(driverVerFile); os.IsNotExist(err) {
		return nil
	}

	dir, err := c.findNvidiaDir()
	if err != nil {
		return err
	}

	if dir == nil {
		return nil
	}

	files, err := filepath.Glob("/dev/nvidia*")
	if err != nil {
		return err
	}


	needToAddDriverDir := false
	if container.Devices.ContainsName(driverDirName) {
		if container.Devices[driverDirName]["source"] != *dir {
			if !doFix {
				return fmt.Errorf("NVidia Binary directory is not pointing to the currently used one")
			}

			//device needs update, remove it and add back later
			err = ubuntu_sdk_tools.RemoveDeviceSync(client, container.Name, driverDirName)
			if err != nil {
				return err
			}
			needToAddDriverDir = true
		}
	} else {
		if !doFix {
			return fmt.Errorf("NVidia Binary directory is not mounted")
		}
		needToAddDriverDir = true
	}

	if (doFix && needToAddDriverDir) {
		err = ubuntu_sdk_tools.AddDeviceSync(
			client, container.Name,
			driverDirName, "disk",
			[]string{fmt.Sprintf("source=%s", *dir), "path=/usr/lib/nvidia-gl", "recursive=true"},
		)
		if err != nil {
			return err
		}
	}

	ldLoaderFile := shared.VarPath("containers", container.Name, "rootfs","etc","ld.so.conf.d","01-nvidia.conf")
	needToWriteLDConf := false
	if _, err := os.Stat(ldLoaderFile); os.IsNotExist(err) {
		needToWriteLDConf = true
	}

	if needToWriteLDConf {
		if !doFix {
			return fmt.Errorf("Need to write the nvidia loader config file")
		}
		fmt.Printf("Writing ld.conf file.\n")
		err = ioutil.WriteFile(ldLoaderFile, []byte("/usr/lib/nvidia-gl\n"),664)
		if err != nil {
			return err
		}
	}

	for _,node := range files {
		if !container.Devices.ContainsName(node) {
			if !doFix {
				return fmt.Errorf("Missing NVidia device")
			}

			err = ubuntu_sdk_tools.AddDeviceSync(
				client, container.Name,
				node, "unix-char",
				[]string{fmt.Sprintf("path=%s", node[1:]),"gid=44"},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *NvidiaFixable) CheckContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, false)
}

func (c *NvidiaFixable) FixContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, true)
}

func (c *NvidiaFixable) Check(client *lxd.Client) error {
	targets, err := ubuntu_sdk_tools.FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		err = c.run(client, &target.Container, false)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *NvidiaFixable) Fix(client *lxd.Client) error {
	fmt.Println("Fixing possible NVidia issues....")
	targets, err := ubuntu_sdk_tools.FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		err = c.run(client, &target.Container, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (*NvidiaFixable) NeedsRoot () bool {
	return false
}

