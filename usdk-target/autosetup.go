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
 * Based on cloud-init (lp:cloud-init):
 * Author: Stéphane Graber
 */
package main

import (
	"fmt"
	"os"
	"launchpad.net/ubuntu-sdk-tools"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/gnuflag"
)

type autosetupCmd struct {
	yes bool
}

func (c *autosetupCmd) usage() string {
	return `Creates a default config for the container backend.

usdk-target autosetup [-y]`
}

func (c *autosetupCmd) flags() {
	gnuflag.BoolVar(&c.yes, "y", false, "Assume yes to all questions.")
}

func (c *autosetupCmd) run(args []string) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("This command needs to run as root")
	}
	if (!c.yes) {
		if(!ubuntu_sdk_tools.GetUserConfirmation("WARNING: This will override existing bridge configurations and restart all your containers, are your sure?")) {
			return fmt.Errorf("Cancelled by user.")
		}
	}

	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		return fmt.Errorf("Could not connect to the LXD server.\n")
	}

	containers, err := client.ListContainers()
	if err != nil {
		return fmt.Errorf("Could not list LXD containers. error: %v", err)
	}

	stoppedContainers := []string{}
	//first let stop the containers
	fmt.Println("Stopping containers:")
	for _, container := range containers {
		if container.StatusCode != 0 && container.StatusCode != shared.Stopped {
			fmt.Printf("Stopping %s .....", container.Name)
			err = ubuntu_sdk_tools.StopContainerSync(client, container.Name)
			if (err != nil) {
				return fmt.Errorf("Could not stop container %s. error: %v.",container.Name, err)
			}
			stoppedContainers = append(stoppedContainers, container.Name)
			fmt.Print(" DONE\n")
		}
	}
	fmt.Println("All containers stopped.")

	fmt.Printf("\nCreating default network bridge .....")
	initCmd := &initializedCmd{}

	err = initCmd.lxdBridgeConfigured(client)
	if err != nil {
		//empty config
		config := map[string]string{}

		bridgeName := "sdkbr0"
		err := client.NetworkCreate(bridgeName, config)
		if err != nil {
			fmt.Print(" FAILED\n")
			return fmt.Errorf("Creating the bridge failed with: %v", err)
		}

		profile, err := client.ProfileConfig("default")
		if err != nil {
			fmt.Print(" FAILED\n")
			return fmt.Errorf("Listing the default profile failed with: %v", err)
		}

		_, eth0ExistsAlready := profile.Devices["eth0"]

		//ok eth0 is already there lets replace it
		if eth0ExistsAlready {
			_, err = client.ProfileDeviceDelete("default", "eth0")
			if err != nil {
				fmt.Print(" FAILED\n")
				return fmt.Errorf("Removing eth0 from default profile failed with: %v", err)
			}
		}

		props := []string{"nictype=bridged", fmt.Sprintf("parent=%s", bridgeName)}
		_, err = client.ProfileDeviceAdd("default", "eth0", "nic", props)
		if err != nil {
			fmt.Print(" FAILED\n")
			return fmt.Errorf("Attaching the bridge to the default profile failed with: %v", err)
		}

		fmt.Println(" DONE")
	} else {
		fmt.Println(" SKIPPED")
	}

	if len(stoppedContainers) > 0 {
		fmt.Println("\nStarting previously stopped containers:")
		for _, container := range stoppedContainers {
			fmt.Printf("Starting %s .....", container)
			err = ubuntu_sdk_tools.BootContainerSync(client, container)
			if (err != nil) {
				fmt.Print(" FAILED\n")
			} else {
				fmt.Print(" DONE\n")
			}
		}

	}

	return nil
}
