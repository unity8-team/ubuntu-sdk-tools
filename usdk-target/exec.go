package main

import (
)
import (
	"fmt"
	"os"
	"os/user"
	"github.com/lxc/lxd/shared/gnuflag"
	"launchpad.net/ubuntu-sdk-tools"
	"syscall"
	"os/exec"
)

type execCmd struct {
	maintMode bool
	container string
	user string
}

func (c *execCmd) usage() string {
	myMode := "exec"
	if (c.maintMode) {
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

	lxc_command, err := exec.LookPath("lxc")
	if err != nil {
		return err
	}

	lxc_args := []string {
		lxc_command, "exec",
		c.container, "--",
		"su",
	}

	if len(args) == 0 {
		lxc_args = append(lxc_args,"-l")
	}

	lxc_args = append(lxc_args, []string {
		"-s", "/bin/bash"}...)

	if (!c.maintMode) {
		lxc_args = append(lxc_args, c.user)
	}

	if len(args) > 0 {
		rcFiles := []string{ "/etc/profile", "$HOME/.profile" }
		cwd, _ := os.Getwd()

		program := ""
		for _,rcfile := range rcFiles {
			program += "test -f "+rcfile+" && . "+rcfile+"; "
		}

		//make sure the working directory is the same
		program += "cd \""+cwd+"\" && "

		//force C locale as QtCreator needs it
		program +=" LC_ALL=C "

		for _,arg := range args {
			program += " "+ubuntu_sdk_tools.QuoteString(arg)
		}

		lxc_args = append(lxc_args, []string {
			"-c", program}...)
	}

	os.Stdout.Sync()
	os.Stderr.Sync()
	err = syscall.Exec(lxc_command, lxc_args, os.Environ())
	fmt.Printf("Error: %v\n", err)
	return nil
}
