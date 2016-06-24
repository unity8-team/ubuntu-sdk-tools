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
	"github.com/lxc/lxd"
	"launchpad.net/ubuntu-sdk-tools"
	"fmt"
	"os"
	"github.com/lxc/lxd/shared"
)

type DevicesFixable struct { }

func (*DevicesFixable) run(client *lxd.Client, container *shared.ContainerInfo, doFix bool) error {

	for devName, dev := range container.Devices {
		var toCheck string = ""
		switch dev["type"] {
		case "disk":
			toCheck,_ = dev["source"]
		case "unix-char":
			_, hasMaj := dev["major"]
			_, hasMin := dev["minor"]
			optional, hasOptional := dev["optional"]

			//do not care about optional devices
			if (hasOptional && optional == "true") {
				continue
			}

			if !(hasMaj && hasMin) {
				toCheck = fmt.Sprintf("/%s", dev["path"])
			}
		}

		if len(toCheck) > 0 {
			if _, err := os.Stat(toCheck); os.IsNotExist(err) {
				if doFix {
					err = ubuntu_sdk_tools.RemoveDeviceSync(client, container.Name, devName)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("Device %s does not exist on the host.", toCheck)
				}
			}
		}
	}
	return nil
}

func (c *DevicesFixable) CheckContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, false)
}

func (c *DevicesFixable) FixContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, true)
}

func (c *DevicesFixable) Check(client *lxd.Client) error {
	fmt.Printf("Checking for broken devices...\n")

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
func (c *DevicesFixable) Fix(client *lxd.Client) error {
	fmt.Println("Checking for and removing broken devices....")
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

func (*DevicesFixable) NeedsRoot () bool {
	return false
}