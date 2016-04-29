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
	"github.com/lxc/lxd/shared/gnuflag"
	"os/user"
	"launchpad.net/ubuntu-sdk-tools"
	"os/exec"
	"syscall"
	"strings"
	"github.com/lxc/lxd"
	"strconv"
)

type registerCmd struct {
	user string
	container string
	createGroups bool
}

func (c *registerCmd) usage() string {
	return `Register a user into the target.

usdk-target register [-u USER] name
`
}

func (c *registerCmd) flags() {
	user, err := userFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not resolve the current user name")
		os.Exit(1)
	}
	c.user = *user
	gnuflag.StringVar(&c.user, "u", c.user, "User to be registered.")
	gnuflag.BoolVar(&c.createGroups, "g", false, "Also try to create the users supplementary groups")
}

func (c *registerCmd) run(args []string) error {
	if (len(args) < 1) {
		fmt.Fprint(os.Stderr, c.usage())
		gnuflag.PrintDefaults()
		return fmt.Errorf("Missing arguments.")
	}
	if (os.Getuid() != 0) {
		return fmt.Errorf("This command needs to run as root")
	}

	c.container = args[0]

	config := ubuntu_sdk_tools.GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the LXD server.\n")
		os.Exit(1)
	}

	return RegisterUserInContainer(client, c.container, &c.user, c.createGroups)
}

func userFromEnv () (*string, error) {

	key := "SUDO_UID"
	env := os.Getenv(key)

	if len(env) == 0 {
		key = "PKEXEC_UID"
		env = os.Getenv(key)
		if len(env) == 0 {
			return nil, nil
		}
		print(env+"\n")
	}

	print(env+"\n")

	user, err := user.LookupId(env)
	if err != nil {
		return nil, fmt.Errorf("Os environment var :%s contains a invalid USER ID. error: %v", key, err)
	}

	return &user.Username, nil
}

func RegisterUserInContainer (client *lxd.Client, containerName string, userName *string, createSupGroups bool) (error) {
	if userName == nil {
		userNameFromEnv, err := userFromEnv()
		if err != nil {
			return err
		}
		//if error is nil and user is nil there was no user in the env
		if userNameFromEnv == nil {
			return fmt.Errorf("Could not determine the user, please use the -a switch.")
		}

		userName = userNameFromEnv
	}

	err := ubuntu_sdk_tools.BootContainerSync(client, containerName)
	if ( err != nil ) {
		return err
	}

	pw, err := ubuntu_sdk_tools.Getpwnam(*userName)
	if (err != nil) {
		return fmt.Errorf("Querying the user entry failed. error: %v", err)
	}

	if pw.Uid == 0 {
		return fmt.Errorf("Registering root is not possible")
	}

	shadow,err := ubuntu_sdk_tools.Getspnam(*userName)
	if (err != nil) {
		return fmt.Errorf("Querying the password entry failed. error: %v", err)
	}

	groups,err := ubuntu_sdk_tools.GetGroups()
	if (err != nil) {
		return fmt.Errorf("Querying the group entry failed. error: %v", err)
	}

	var requiredGroups []ubuntu_sdk_tools.GroupEntry
	for _, group := range groups {
		if group.Gid == pw.Gid {
			requiredGroups = append(requiredGroups, group)
			if (createSupGroups) {
				continue
			} else {
				break
			}
		}
		if (createSupGroups) {
			for _, member := range group.Members {
				if member == *userName {
					requiredGroups = append(requiredGroups, group)
					break
				}
			}
		}
	}

	err = ubuntu_sdk_tools.AddDeviceSync(client,containerName,
		fmt.Sprintf("home_of_%s", *userName),
		"disk",
		[]string{fmt.Sprintf("source=%s",pw.Dir), fmt.Sprintf("path=%s",pw.Dir)})
	if (err != nil) {
		return fmt.Errorf("Failed to mount home directory of the user: %s. error: %v", *userName, err)
	}

	print ("Creating groups\n")
	var supplGroups []string
	for _, group := range requiredGroups {
		mustWork := group.Gid == pw.Gid

		fmt.Printf("Creating group %s\n", group.Name)

		cmd := exec.Command("lxc", "exec", containerName, "--", "groupadd", "-g",  strconv.FormatUint(uint64(group.Gid),10), group.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err := cmd.Wait(); err != nil {
			print ("GroupAdd returned error\n")
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					//exit code of 9 means the group exists already
					//which we will treat as success
					if status.ExitStatus() != 9 {
						if mustWork {
							return fmt.Errorf("Could not create primary group")
						}
						continue
					}
				}
			} else {
				return fmt.Errorf("Failed to add the group %s. error: %v", group.Name, err)
			}

			if !mustWork {
				supplGroups = append(supplGroups, group.Name)
			}
		}
	}

	fmt.Printf("Creating user %s\n", pw.LoginName)

	command := []string {
		"exec", containerName, "--",
		"useradd", "--no-create-home",
		"-u", strconv.FormatUint(uint64(pw.Uid), 10),
		"--gid", strconv.FormatUint(uint64(pw.Gid), 10),
		"--home-dir", pw.Dir,
		"-s", "/bin/bash",
		"-p", shadow.Sp_pwdp,
	}

	if len(supplGroups) > 0 {
		command = append(command, "--groups",strings.Join(supplGroups, ","))
	}

	command = append(command,pw.LoginName)

	cmd := exec.Command("lxc", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}