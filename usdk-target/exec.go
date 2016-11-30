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

import ()
import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared/gnuflag"
	"launchpad.net/ubuntu-sdk-tools"
	"os"
	"os/user"
)

type execCmd struct {
	maintMode bool
	container string
	user      string
}

func (c *execCmd) usage() string {
	myMode := "exec"
	if c.maintMode {
		myMode = "maint"
	}

	return fmt.Sprintf(`Executes a command in the container.

usdk-target %s container [command]`, myMode)
}

func (c *execCmd) flags() {
	user, err := user.Current()
	if err == nil {
		c.user = user.Username
	}

	gnuflag.StringVar(&c.user, "u", c.user, "Username to login before executing the command.")
}

func (c *execCmd) run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, c.usage())
		os.Exit(1)
	}

	c.container = args[0]
	args = args[1:]

	config := ubuntu_sdk_tools.GetConfigOrDie()
	d, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		return err
	}

	lxc_args := []string{
		"su",
	}

	if len(args) == 0 {
		lxc_args = append(lxc_args, "-l")
	}

	lxc_args = append(lxc_args, []string{
		"-s", "/bin/bash"}...)

	if !c.maintMode {
		lxc_args = append(lxc_args, c.user)
	}

	if len(args) > 0 {
		rcFiles := []string{"/etc/profile", "$HOME/.profile"}
		cwd, _ := os.Getwd()

		program := ""
		for _, rcfile := range rcFiles {
			program += "test -f " + rcfile + " && . " + rcfile + "; "
		}

		//make sure the working directory is the same
		program += "cd \"" + cwd + "\" && "

		//force C locale as QtCreator needs it
		program += " LC_ALL=C "

		for _, arg := range args {
			program += " " + ubuntu_sdk_tools.QuoteString(arg)
		}

		lxc_args = append(lxc_args, []string{
			"-c", program}...)
	}

	os.Stdout.Sync()
	os.Stderr.Sync()
	// Ensure the container's running first
	err = ubuntu_sdk_tools.BootContainerSync(d, c.container)
	if err != nil {
		return err
	}
	controlHandler := func(*lxd.Client, *websocket.Conn) {

	}
	_, err = d.Exec(c.container, lxc_args, nil, os.Stdin, os.Stdout, os.Stderr, controlHandler, 0, 0)
	if err != nil {
		return err
	}
	return nil
}
