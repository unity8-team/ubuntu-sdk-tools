package main

import (
)
import (
	"fmt"
	"os"
	"github.com/lxc/lxd/shared"
)

type rootfsCmd struct {
}

func (c *rootfsCmd) usage() string {
	return `Shows the path to the root filesystem of a container.

usdk-target rootfs container`
}

func (c *rootfsCmd) flags() {
}

func (c *rootfsCmd) run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, c.usage())
		os.Exit(1)
	}
	fmt.Printf(ContainerRootfs(args[0]))

	return nil
}

func ContainerRootfs (container string) (string) {
	return shared.VarPath("containers", container, "rootfs")
}
