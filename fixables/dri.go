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
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"launchpad.net/ubuntu-sdk-tools"
	"os"
	"path/filepath"
)

type DRIFixable struct{}

func (*DRIFixable) run(client *lxd.Client, container *shared.ContainerInfo, doFix bool) error {
	// FIXME: dri device isn't accessible under confinement
	if os.Getenv("SNAP") != "" {
		fmt.Fprintf(os.Stderr, "Skipping adding /dev/dri/card*\n")
		return nil
	}

	files, err := filepath.Glob("/dev/dri/card*")
	if err != nil {
		return err
	}

	for _, node := range files {
		if !container.Devices.ContainsName(node) {
			if doFix {
				muh := container.Name
				err = ubuntu_sdk_tools.AddDeviceSync(
					client, muh,
					node, "unix-char",
					[]string{fmt.Sprintf("path=%s", node[1:]), "gid=44"},
				)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Container is missing device node: %s", node)
			}
		}

	}

	return nil
}

func (c *DRIFixable) CheckContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, false)
}

func (c *DRIFixable) FixContainer(client *lxd.Client, container string) error {
	info, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	return c.run(client, info, true)
}

func (c *DRIFixable) Check(client *lxd.Client) error {
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
func (c *DRIFixable) Fix(client *lxd.Client) error {
	fmt.Println("Fixing possible DRI devices....")
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

func (*DRIFixable) NeedsRoot() bool {
	return false
}
