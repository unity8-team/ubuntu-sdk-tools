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
)
import (
	"fmt"
	"os"
	"launchpad.net/ubuntu-sdk-tools"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared/gnuflag"
)

const (
	ERR_NO_ACCESS    = 255
	ERR_NEEDS_FIXING = 254
	ERR_NO_BRIDGE    = 253
	//ERR_UNKNOWN      = 200
)

type initializedCmd struct {
	ignoreBridgeCheck bool
}

func (c *initializedCmd) usage() string {
	return `Checks if the container backend is setup correctly.

usdk-target initialized`
}

func (c *initializedCmd) flags() {
	gnuflag.BoolVar(&c.ignoreBridgeCheck, "b", false, "Do not check for lxd bridge")
}

func (c *initializedCmd) run(args []string) error {
	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the container backend.\n")
		os.Exit(ERR_NO_ACCESS)
	}

	_, err = client.ServerStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not talk to the container backend.\n")
		os.Exit(ERR_NO_ACCESS)
	}

	if !c.ignoreBridgeCheck {
		err := c.lxdBridgeConfigured(client)
		if (err != nil) {
			os.Exit(ERR_NO_BRIDGE)
		}
		fmt.Println("LXD bridge is configured with a subnet.")
	} else {
		fmt.Println("Skipping bridge check.")
	}

	for _,fixable := range fixable_set {
		fixableErr := fixable.Check(client)
		if fixableErr != nil {
			fmt.Printf("Error: %v\n", fixableErr)
			os.Exit(ERR_NEEDS_FIXING)
		}
	}

	fmt.Println("Container backend is ready.")
	return nil
}

func (c *initializedCmd) lxdBridgeConfigured (client *lxd.Client) (error) {
	netConfList, err := client.ListNetworks()
	if err != nil {
		return err
	}

	for _,netConf := range netConfList {
		if !netConf.Managed || netConf.Type != "bridge" {
			continue
		}

		if addr, ok := netConf.Config["ipv4.address"]; !ok || addr == "" {
			continue
		}

		//compatible bridge found
		return nil
	}

	// we found nothing
	return fmt.Errorf("lxd-bridge not configured")
}
