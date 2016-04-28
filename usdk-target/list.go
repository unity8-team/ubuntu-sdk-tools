package main

import (
	"github.com/lxc/lxd"
	"encoding/json"
	"launchpad.net/ubuntu-sdk-tools"
)

type listCmd struct {
}

type clickContainer struct {
	Name string `json:"name"`
	Architecture string `json:"architecture"`
	Framework string `json:"framework"`
}

func (c *listCmd) usage() string {
	return (
		`Lists the existing SDK build targets.

usdk-wrapper list`)
}

func (c *listCmd) flags() {
}

func (c *listCmd) run(args []string) error {

	config := ubuntu_sdk_tools.GetConfigOrDie()
	d, err := lxd.NewClient(config, config.DefaultRemote)
	if err != nil {
		return err
	}

	ctslist, err := d.ListContainers()
	if err != nil {
		return err
	}

	clickTargets := []clickContainer{}

	for _, cInfo := range ctslist {
		cConf := cInfo.Config
		clickArch, ok := cConf["user.click-architecture"]
		if !ok {
			continue
		}

		clickFW, ok := cConf["user.click-framework"]
		if !ok {
			continue
		}

		clickTargets = append(clickTargets, clickContainer{Name:cInfo.Name, Architecture: clickArch, Framework: clickFW})
	}

	data, err := json.Marshal(clickTargets)
	if err != nil{
		return err
	}

	print(string(data))

	return nil
}