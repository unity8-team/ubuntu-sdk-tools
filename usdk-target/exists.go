package main

import (
)
import (
	"fmt"
	"os"
	"github.com/lxc/lxd"
	"launchpad.net/ubuntu-sdk-tools"
)

type existsCmd struct {
}

func (c *existsCmd) usage() string {
	return `Checks if a container exists.

usdk-target exists container`
}

func (c *existsCmd) flags() {
}

func (c *existsCmd) run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, c.usage())
		os.Exit(1)
	}

	config := ubuntu_sdk_tools.GetConfigOrDie()
	d, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		return err
	}

	allContainers, err := d.ListContainers()
	if err != nil {
		return fmt.Errorf("Could not query the containers. error: %v.\n", err)
	}

	for _, cont := range allContainers {
		if cont.Name == args[0] {
			println("Container exists")
			return nil
		}
	}

	return fmt.Errorf("Container not found")
}
