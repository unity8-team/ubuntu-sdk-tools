package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"bufio"
	"bytes"
)

type helpCmd struct {
}

func (c *helpCmd) usage() string {
	return "Presents details on how to use usdk-wrapper."
}

func (c *helpCmd) flags() {
}

func (c *helpCmd) run(args []string) error {
	if len(args) > 0 {
		for _, name := range args {
			cmd, ok := commands[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "error: unknown command: %s\n", name)
			} else {
				fmt.Fprintf(os.Stderr, cmd.usage()+"\n")
			}
		}
		return nil
	}

	fmt.Println("Usage: usdk-wrapper [subcommand] [options]")
	fmt.Println("Available commands:")
	var names []string
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := commands[name]
		fmt.Printf("\t%-10s - %s\n", name, c.summaryLine(cmd.usage()))
	}

	fmt.Println("Environment:")
	fmt.Println("  LXD_CONF           " + "Path to an alternate client configuration directory.")
	fmt.Println("  LXD_DIR            " + "Path to an alternate server directory.")
	return nil
}

// summaryLine returns the first line of the help text. Conventionally, this
// should be a one-line command summary, potentially followed by a longer
// explanation.
func (c *helpCmd) summaryLine(usage string) string {
	usage = strings.TrimSpace(usage)
	s := bufio.NewScanner(bytes.NewBufferString(usage))
	if s.Scan() {
		if len(s.Text()) > 1 {
			return s.Text()
		}
	}
	return "Missing summary."
}
