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
	"os"
	"github.com/lxc/lxd"
	"launchpad.net/ubuntu-sdk-tools"
)

type setCmd struct {
}

func (c *setCmd) usage() string {
	return (
	`Change container flags.

usdk-target set <container> upgrades-enabled	Flag container for automatic updgrade checks (from the SDK IDE)
usdk-target set <container> upgrades-disabled	Flag container for exclusion from automatic updgrade checks (from the SDK IDE)`)
}

func (c *setCmd) flags() {
}

func (c *setCmd) run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Wrong number of arguments")
	}

	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the container backend.\n")
		os.Exit(ERR_NO_ACCESS)
	}

	switch args[1] {

	case "upgrades-enabled":
		err = client.SetContainerConfig(args[0], ubuntu_sdk_tools.TargetUpgradesConfig, "true")
	case "upgrades-disabled":
		err = client.SetContainerConfig(args[0], ubuntu_sdk_tools.TargetUpgradesConfig, "false")
	default:
		return fmt.Errorf("Unknown command: %s", args[1])

	}

	return err
}