package main
import (
	"github.com/lxc/lxd"
	"os"
	"io"
	"bytes"
	"regexp"
	"strings"
	"fmt"
	"path/filepath"
	"os/user"
	"os/signal"
	"syscall"
	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
	"path"
)

var container string
var configPath string
var _find_unsafe = regexp.MustCompile("[^\\w@%+=:,./-]")

func mapAndWrite (line *bytes.Buffer, out io.WriteCloser) {
	paths := []string{"var","bin","boot","dev","etc","lib","lib64","media","mnt","opt","proc","root","run","sbin","srv","sys","usr"}
	in := string(line.Bytes())
	for _,path := range paths {
		re := regexp.MustCompile("(^|[^\\w+]|\\s+|-\\w)\\/("+path+")")
		in = re.ReplaceAllString(in, "$1/var/lib/lxd/containers/"+container+"/rootfs/$2")
	}
	out.Write([]byte(in))
}

func quoteString (s string) string{
	//Return a shell-escaped version of the string *s*.
	if len(s) == 0 {
		return "''"
	}

	if !_find_unsafe.MatchString(s) {
		return s
	}


	// use single quotes, and put single quotes into double quotes
	// the string $'b is then quoted as '$'"'"'b'

	return "'" + strings.Replace(s, "'", "'\"'\"'", -1) + "'"
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

	configDir := "$HOME/.config/lxc"
	if os.Getenv("LXD_CONF") != "" {
		configDir = os.Getenv("LXD_CONF")
	}
	configPath = os.ExpandEnv(path.Join(configDir, "config.yml"))

	config, err := lxd.LoadConfig(configPath)
	if err != nil {
		print("Could not load LXC config")
		os.Exit(1)
	}

	cl, err := lxd.NewClient(config, "local")
	if err != nil {
		print("Could not connect to the LXD server")
		os.Exit(1)
	}

	//figure out the container we should execute the command in
	//the parent directories name is supposed to be named like it
	toolpath,err := filepath.Abs(os.Args[0])
	if err != nil {
		print("Could not resolve the absolute pathname of the tool")
		os.Exit(1)
	}

	container = filepath.Base(filepath.Dir(toolpath))

	//we mirror the current user into the LXD container
	user, err := user.Current()
	if err != nil {
		print("Could not resolve the current user name")
		os.Exit(1)
	}

	//build the command, sourcing the dotfiles to get a decent shell
	args := []string{}
	args = append(args, filepath.Base(os.Args[0]))
	args = append(args, os.Args[1:]...)

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
		program += " "+quoteString(arg)
	}

	stdout_r, stdout_w := io.Pipe()
	stderr_r, stderr_w := io.Pipe()

	go mapFunc(stdout_r, os.Stdout)
	go mapFunc(stderr_r, os.Stderr)

	controlSocketHandler := func (d *lxd.Client, control *websocket.Conn) {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		for {
			sig := <-ch
			cl.Exec(container, []string{
				"/bin/bash",
				"-c",
				fmt.Sprintf("kill -%d `cat %s`", sig, pidfile),
			}, map[string]string{}, os.Stdin, nil, nil, nil, 0, 0)
		}
	}

	code, err := cl.Exec(container,
		[]string{"su", user.Username, "-s", "/bin/bash", "-c", "/bin/bash", "-c", program },
		map[string]string{},
		os.Stdin,
		stdout_w,
		stderr_w,
		controlSocketHandler, 0, 0)

	stdout_r.Close()
        stdout_w.Close()
	stderr_r.Close()
        stderr_w.Close()

	cl.Exec(container, []string{
			"rm",
			pidfile,
		},
		map[string]string{},
		os.Stdin, nil, nil, nil, 0, 0)

	if code != 0 {
		os.Exit(code)
	}

        if err != nil || code != 0 {
		os.Exit(1)
	}

	os.Exit(0)
}
