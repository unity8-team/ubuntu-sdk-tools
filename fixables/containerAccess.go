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
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd"
	"os"
	"fmt"
	"path/filepath"
	"launchpad.net/ubuntu-sdk-tools"
)

type ContainerAccess struct { }
func (*ContainerAccess) run(client *lxd.Client, container string, doFix bool) error {
	targetPath := shared.VarPath("containers", container)
	fi, err := os.Lstat(targetPath)
	if err != nil {
		return fmt.Errorf("Failed to query container access permissions\n",err)
	}

	if fi.Mode() & os.ModeSymlink == os.ModeSymlink {
		targetPath, err = filepath.EvalSymlinks(targetPath)
		if err != nil {
			return fmt.Errorf("Failed to read rootfs link. error: %v.\n",err)
		}

		fi, err = os.Lstat(targetPath)
		if err != nil {
			return fmt.Errorf("Failed to query container access permissions\n",err)
		}
	}

	if fi.Mode() != os.ModeDir | ubuntu_sdk_tools.LxdContainerPerm {
		if !doFix {
			fmt.Printf("Wrong directory permissions. Container rootfs of %s is not accessible.", container)
		} else {
			err = os.Chmod(targetPath, ubuntu_sdk_tools.LxdContainerPerm)
			if err != nil {
				fmt.Printf("Failed to make container readable. error: %v.\n",err)
			}
		}

	}
	return nil
}

func (c *ContainerAccess) CheckContainer(client *lxd.Client, container string) error {
	return c.run(client, container, false)
}

func (c *ContainerAccess) FixContainer(client *lxd.Client, container string) error {
	return c.run(client, container, true)
}

func (c *ContainerAccess) Check(client *lxd.Client) error {

	fmt.Println("Checking if containers are accessible")
	targets, err := ubuntu_sdk_tools.FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		err := c.run(client, target.Name, false)
		if err != nil {
			return err
		}
	}

	fmt.Println("All containers are accessible.")
	return nil
}

func (c *ContainerAccess) Fix(client *lxd.Client) error {
	fmt.Println("Fixing possible container permission problems....")
	targets, err := ubuntu_sdk_tools.FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		err := c.run(client, target.Name, true)
		if err != nil {
			return err
		}
	}

	fmt.Println("All containers are accessible.")
	return nil
}

func (*ContainerAccess) NeedsRoot () bool {
	return true
}
