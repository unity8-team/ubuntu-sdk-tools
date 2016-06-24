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
	"launchpad.net/ubuntu-sdk-tools/fixables"
)

var fixable_set = []fixables.Fixable{
	&fixables.ContainerAccess{},
	&fixables.DevicesFixable{},
	&fixables.DRIFixable{},
	&fixables.NvidiaFixable{},
}

type autofixCmd struct {
}

func (c *autofixCmd) usage() string {
	return `Automatically fixes problems in the container backends.`
}

func (c *autofixCmd) flags() {
}

func (c *autofixCmd) run(args []string) error {
	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the container backend.\n")
		os.Exit(ERR_NO_ACCESS)
	}

	for _, fixable := range fixable_set {
		err = fixable.Fix(client)
		if err != nil {
			return err
		}
	}

	//TODO reboot all containers

	return nil
}