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
package ubuntu_sdk_tools

import (
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"path"
	"os"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const LxdBridgeFile = "/etc/default/lxd-bridge"
var globConfig *lxd.Config = nil

func EnsureLXDInitializedOrDie() {
	config := GetConfigOrDie()
	client, err := lxd.NewClient(config, config.DefaultRemote)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to LXD. Error: %v\n", err)
		os.Exit(1)
	}

	/*[26.04 16:33:38] <stgraber> zbenjamin: ok, so about that user.network_mode key.
	 You may want to query it twice if it's the first LXD query you do as user.network_mode
	 is set by the init script right after LXD startup, which means that you may be getting the previous (unset)
	 value if your query is the one which starts LXD.*/
	_, _ = client.GetProfileConfig("default")
	defaultProfile, err := client.GetProfileConfig("default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not query status from LXD. Error: %v\n", err)
		os.Exit(1)
	}

	//check if network mode is there and if it is that its not link-local
	networkMode, ok := defaultProfile["user.network_mode"]
	if ok {
		if networkMode == "link-local" {
			fmt.Fprintf(os.Stderr, "LXD is not set up correctly, please run lxd init to configure a subnet")
			os.Exit(255)
		}
	}


	//if we reached this place lets register a new remote
	defaultImageRemote := "https://sdk-images.canonical.com"
	defaultRemoteName  := "ubuntu-sdk-images"
	remotes := config.Remotes
	sdkRem, ok := remotes[defaultRemoteName]
	if ok {
		if sdkRem.Addr == defaultImageRemote {
			return
		} else {
			cmd := exec.Command("lxc", "remote", "remove", defaultRemoteName)
			err = cmd.Run()
			if (err != nil) {
				fmt.Fprintf(os.Stderr, "Could not remove the remote "+defaultRemoteName+". error: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please remove it manually.\n", err)
				os.Exit(1)
			}
		}
	}

	cmd := exec.Command("lxc", "remote", "add", "ubuntu-sdk-images", defaultImageRemote, "--accept-certificate", "--protocol=simplestreams")
	err = cmd.Run()
	if (err != nil) {
		fmt.Fprintf(os.Stderr, "Could not register remote. error: %v\n", err)
		os.Exit(1)
	}

	//make sure config is loaded again
	globConfig = nil
}

func GetConfigOrDie ()  (*lxd.Config) {

	if globConfig != nil {
		return globConfig
	}

	configDir := "$HOME/.config/lxc"
	if os.Getenv("LXD_CONF") != "" {
		configDir = os.Getenv("LXD_CONF")
	}
	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))

	globConfig, err := lxd.LoadConfig(configPath)
	if err != nil {
		log.Fatal("Could not load LXC config")
	}

	certf := globConfig.ConfigPath("client.crt")
	keyf := globConfig.ConfigPath("client.key")

	if !shared.PathExists(certf) || !shared.PathExists(keyf) {
		fmt.Fprintf(os.Stderr, "Generating a client certificate. This may take a minute...\n")

		err = shared.FindOrGenCert(certf, keyf)
		if err != nil {
			log.Fatal("Could not generate client certificates.\n")
			os.Exit(1)
		}

		if shared.PathExists("/var/lib/lxd/") {
			fmt.Fprintf(os.Stderr, "If this is your first time using LXD, you should also run: sudo lxd init\n\n")
		}
	}

	return globConfig
}

func BootContainerSync (client *lxd.Client, name string) error {
	current, err := client.ContainerInfo(name)
	if err != nil {
		return err
	}

	action := shared.Start

	if current.StatusCode == shared.Running {
		return nil
	}

	// "start" for a frozen container means "unfreeze"
	if current.StatusCode == shared.Frozen {
		action = shared.Unfreeze
	}


	resp, err := client.Action(name, action, 10, false, false)
	if err != nil {
		return err
	}

	if resp.Type != lxd.Async {
		return fmt.Errorf("bad result type from action")
	}

	if err := client.WaitForSuccess(resp.Operation); err != nil {
		return fmt.Errorf("%s\nTry `lxc info --show-log %s` for more info", err, name)
	}
	return nil
}

func StopContainerSync  (client *lxd.Client, container string) error {
	ct, err := client.ContainerInfo(container)
	if err != nil {
		return err
	}

	if ct.StatusCode != 0 && ct.StatusCode != shared.Stopped {
		resp, err := client.Action(container, shared.Stop, -1, true, false)
		if err != nil {
			return err
		}

		if resp.Type != lxd.Async {
			return fmt.Errorf("bad result type from action")
		}

		if err := client.WaitForSuccess(resp.Operation); err != nil {
			return fmt.Errorf("%s\nTry `lxc info --show-log %s` for more info", err, container)
		}

		if ct.Ephemeral == true {
			return nil
		}
	}
	return nil
}

func AddDeviceSync (client *lxd.Client, container, devname, devtype string, props []string) error{
	fmt.Printf("Adding device %s\n",devname)
	resp, err := client.ContainerDeviceAdd(container, devname, devtype, props)
	if err != nil {
		return err
	}

	err = client.WaitForSuccess(resp.Operation)
	if err == nil {
		fmt.Printf("Device %s added to %s\n", devname, container)
	}
	return err
}

func RemoveContainerSync(client *lxd.Client, container string) (error){

	err := StopContainerSync(client, container)
	if err != nil {
		return err
	}

	resp, err := client.Delete(container)
	if err != nil {
		return err
	}

	return client.WaitForSuccess(resp.Operation)
}

func GetUserConfirmation(question string) (bool) {
	var response string
	responses := map[string]bool{
		"y": true, "yes": true,
		"n": false, "no": false,
	}

	ok := false
	answer := false
	for !ok {
		fmt.Print(question+" (yes/no): ")
		_, err := fmt.Scanln(&response)
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(response)
		answer, ok = responses[response]
	}

	return answer
}