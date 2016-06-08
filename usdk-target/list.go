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
	"github.com/lxc/lxd"
	"encoding/json"
	"launchpad.net/ubuntu-sdk-tools"
	"fmt"
)

type listCmd struct {
}

func (c *listCmd) usage() string {
	return (
		`Lists the existing SDK build targets.

usdk-wrapper list`)
}

func (c *listCmd) flags() {
}

func (c *listCmd) run(args []string) error {

	config := ubuntu_sdk_tools.GetConfigOrDie()
	d, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		return err
	}

	clickTargets, err := ubuntu_sdk_tools.FindClickTargets(d)
	if err != nil {
		return nil
	}

	data, err := json.Marshal(clickTargets)
	if err != nil{
		return err
	}

	fmt.Printf("%s\n", data)
	return nil
}