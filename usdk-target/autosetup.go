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
	"github.com/lxc/lxd/shared/gnuflag"
	"bytes"
	"fmt"
	"strings"
	"strconv"
	"os/exec"
	"io/ioutil"
	"os"
	"launchpad.net/ubuntu-sdk-tools"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
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

	subnet, err := c.detectSubnet()
	if err != nil  {
		return err
	}

	err = c.editLXDBridgeFile(subnet)
	if err != nil {
		return err
	}

	fmt.Print("\nRestarting services .....")
	cmd := exec.Command("bash", "-c", "service lxd stop && service lxd-bridge stop && service lxd-bridge start && service lxd start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Restarting the LXD service failed. error: %v", err)
	}
	fmt.Print(" DONE\n")

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

func (c *autosetupCmd) detectSubnet() (string, error) {
	used := make([]int, 0)

	ipAddrOutput, err := exec.Command("ip", "addr", "show").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(ipAddrOutput), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		columns := strings.Split(trimmed, " ")

		if len(columns) < 2 {
			return "", fmt.Errorf("invalid ip addr output line %s", line)
		}

		if columns[0] != "inet" {
			continue
		}

		addr := columns[1]
		if !strings.HasPrefix(addr, "10.0.") {
			continue
		}

		tuples := strings.Split(addr, ".")
		if len(tuples) < 4 {
			return "", fmt.Errorf("invalid ip addr %s", addr)
		}

		subnet, err := strconv.Atoi(tuples[2])
		if err != nil {
			return "", err
		}

		used = append(used, subnet)
	}

	curr := 1
	for {
		isUsed := false
		for _, subnet := range used {
			if subnet == curr {
				isUsed = true
				break
			}
		}
		if !isUsed {
			break
		}

		curr++
		if curr > 254 {
			return "", fmt.Errorf("No valid subnet available")
		}
	}

	return fmt.Sprintf("%d", curr), nil
}

func (c *autosetupCmd) editLXDBridgeFile(subnet string) error {
	buffer := bytes.Buffer{}

	f, err := os.OpenFile(ubuntu_sdk_tools.LxdBridgeFile,  os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	input := string(data)
	newValues := map[string]string{
		"USE_LXD_BRIDGE":      "true",
		"EXISTING_BRIDGE":     "",
		"LXD_BRIDGE":          "lxdbr0",
		"LXD_IPV4_ADDR":       fmt.Sprintf("10.0.%s.1", subnet),
		"LXD_IPV4_NETMASK":    "255.255.255.0",
		"LXD_IPV4_NETWORK":    fmt.Sprintf("10.0.%s.1/24", subnet),
		"LXD_IPV4_DHCP_RANGE": fmt.Sprintf("10.0.%s.2,10.0.%s.254", subnet, subnet),
		"LXD_IPV4_DHCP_MAX":   "253",
		"LXD_IPV4_NAT":        "true",
		"LXD_IPV6_PROXY":      "false",
	}
	found := map[string]bool{}

	for _, line := range strings.Split(input, "\n") {
		out := line

		if !strings.HasPrefix(line, "#") {
			for prefix, value := range newValues {
				if strings.HasPrefix(line, prefix + "=") {
					out = fmt.Sprintf(`%s="%s"`, prefix, value)
					found[prefix] = true
					break
				}
			}
		}

		buffer.WriteString(out)
		buffer.WriteString("\n")
	}

	for prefix, value := range newValues {
		if !found[prefix] {
			buffer.WriteString(prefix)
			buffer.WriteString("=")
			buffer.WriteString(value)
			buffer.WriteString("\n")
			found[prefix] = true // not necessary but keeps "found" logically consistent
		}
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = f.WriteString(buffer.String())
	if err != nil {
		return err
	}
	return nil
}