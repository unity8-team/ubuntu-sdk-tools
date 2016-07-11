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
	"github.com/lxc/lxd"
	"os"
	"io"
	"bytes"
	"regexp"
	"fmt"
	"path/filepath"
	"os/user"
	"os/signal"
	"syscall"
	"github.com/pborman/uuid"
	"launchpad.net/ubuntu-sdk-tools"
	"path"
)

var container string

func mapAndWrite (line *bytes.Buffer, out io.WriteCloser) {
	paths := []string{"var","bin","boot","dev","etc","lib","lib64","media","mnt","opt","proc","root","run","sbin","srv","sys","usr"}
	in := string(line.Bytes())
	for _,path := range paths {
		re := regexp.MustCompile("(^|[^\\w+]|\\s+|-\\w)\\/("+path+")")
		in = re.ReplaceAllString(in, "$1/var/lib/lxd/containers/"+container+"/rootfs/$2")
	}
	out.Write([]byte(in))
}

func mapFunc (in *io.PipeReader, output io.WriteCloser) {
	readBuf := make([]byte, 1)
	var lineBuf bytes.Buffer
	defer in.Close()
	for {
		n, err := in.Read(readBuf)

		if err != nil {
			break
		}

		if n > 0 {
			lineBuf.Write(readBuf)
			if (readBuf[0] == byte('\n')) {
				mapAndWrite(&lineBuf, output)
				lineBuf.Truncate(0)
			}
		}
	}

	if (lineBuf.Len() > 0) {
		mapAndWrite(&lineBuf, output)
	}
}

func main()  {
	config := ubuntu_sdk_tools.GetConfigOrDie()
	cl, err := lxd.NewClient(config, "local")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to the LXD server")
		os.Exit(1)
	}

	//figure out the container we should execute the command in
	//the parent directories name is supposed to be named like it
	toolpath,err := filepath.Abs(os.Args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not resolve the absolute pathname of the tool")
		os.Exit(1)
	}

	container = filepath.Base(filepath.Dir(toolpath))

	err = ubuntu_sdk_tools.BootContainerSync(cl, container)
	if (err != nil) {
		fmt.Fprintf(os.Stderr, "Error while starting the container: %v\n",err)
		os.Exit(1)
	}

	//we mirror the current user into the LXD container
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not resolve the current user name")
		os.Exit(1)
	}

	cmdName := filepath.Base(os.Args[0])
	cmdArgs := os.Args[1:]

	if (cmdName == "cmake") {
		helpMode := false
		for _, opt := range cmdArgs {
			if (opt == "help") {
				helpMode = true
				break
			}
		}

		if (!helpMode) {
			cwd, _ := os.Getwd()
			if _, err := os.Stat(path.Join(cwd, "CMakeCache.txt")); err == nil {
				fmt.Printf("-- Removing build artifacts\n")
				_= os.RemoveAll(path.Join(cwd, "CMakeFiles"))
				_= os.Remove(path.Join(cwd, "CMakeCache.txt"))
				_= os.Remove(path.Join(cwd, "cmake_install.cmake"))
				_= os.Remove(path.Join(cwd, "Makefile"))
			}
		}
	}

	//build the command, sourcing the dotfiles to get a decent shell
	args := []string{}
	args = append(args, cmdName)
	args = append(args, cmdArgs...)

	//until LXD supports sending signals to processes we need to have a pidfile
	u1 := uuid.NewUUID()
	pidfile := fmt.Sprintf("/tmp/%x.pid", u1)

	rcFiles := []string{ "/etc/profile", "$HOME/.profile" }
	cwd, _ := os.Getwd()

	program := ""
	for _,rcfile := range rcFiles {
		program += "test -f "+rcfile+" && . "+rcfile+"; "
	}

	//make sure the working directory is the same
	program += "cd \""+cwd+"\" && "

	//write the current shells PID into the pidfile
	program += fmt.Sprintf("echo $$ > %s; ", pidfile)

	//force C locale as QtCreator needs it
	program +=" LC_ALL=C exec"

	for _,arg := range args {
		program += " "+ubuntu_sdk_tools.QuoteString(arg)
	}

	stdout_r, stdout_w := io.Pipe()
	stderr_r, stderr_w := io.Pipe()

	go mapFunc(stdout_r, os.Stdout)
	go mapFunc(stderr_r, os.Stderr)

	go func () {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		for {
			sig := <-ch
			cl.Exec(container, []string{
				"/bin/bash",
				"-c",
				fmt.Sprintf("kill -%d -$(ps -o pgid= `cat %s` | grep -o '[0-9]*')", sig, pidfile),
			}, map[string]string{}, os.Stdin, nil, nil, nil, 0, 0)
		}
	} ()

	code, err := cl.Exec(container,
		[]string{"su", user.Username, "-s", "/bin/bash", "-c", "/bin/bash", "-c", program },
		map[string]string{},
		os.Stdin,
		stdout_w,
		stderr_w,
		nil, 0, 0)

	stdout_r.Close()
        stdout_w.Close()
	stderr_r.Close()
        stderr_w.Close()

	//since the pidfile is created in /tmp and /tmp is mounted into the container
	//we can just delete the local file
	err = os.Remove(pidfile)

	if code != 0 {
		os.Exit(code)
	}

        if err != nil || code != 0 {
		os.Exit(1)
	}

	os.Exit(0)
}
