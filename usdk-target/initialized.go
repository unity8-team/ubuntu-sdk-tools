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
	"io/ioutil"
	"strings"
	"launchpad.net/ubuntu-sdk-tools"
)

type initializedCmd struct {
}

func (c *initializedCmd) usage() string {
	return `Checks if the container backend is setup correctly.

usdk-target initialized`
}

func (c *initializedCmd) flags() {
}

func (c *initializedCmd) run(args []string) error {
	err := c.lxdBridgeConfigured()
	if (err != nil) {
		return err
	}

	fmt.Println("Container backend is ready.")
	return nil
}

func (c *initializedCmd) lxdBridgeConfigured () (error) {
	f, err := os.Open(ubuntu_sdk_tools.LxdBridgeFile)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	requiredValues := map[string]string {
		"USE_LXD_BRIDGE": "",
		"LXD_IPV4_ADDR": "",
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}

		dataSet := strings.Split(line, "=")
		if len(dataSet) != 2 {
			continue
		}

		prefix := strings.TrimSpace(dataSet[0])
		data   := strings.TrimSpace(dataSet[1])
		data   = strings.Trim(data,"\"")

		_, ok := requiredValues[prefix]
		if ok {
			fmt.Printf("Key %v has value %v\n",prefix, data)
			requiredValues[prefix] = data
		}

	}

	if requiredValues["USE_LXD_BRIDGE"] != "true" || requiredValues["LXD_IPV4_ADDR"] == ""{
		return fmt.Errorf("lxd-bridge not configured")
	}

	return nil
}
