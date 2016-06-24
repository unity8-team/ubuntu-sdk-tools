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
)
import (
	"github.com/lxc/lxd"
	"launchpad.net/ubuntu-sdk-tools"
	"fmt"
	"encoding/json"
	"time"
	"strings"
)

type imageDesc struct {
	Alias string `json:"alias"`
	Fingerprint string `json:"fingerprint"`
	Description string `json:"desc"`
	Arch string `json:"arch"`
	Size int64 `json:"size"`
	UploadDate time.Time `json:"uploadDate"`
}

type imagesCmd struct {
}

func (c *imagesCmd) usage() string {
	return `Shows the available Ubuntu SDK images.

usdk-target images`
}

func (c *imagesCmd) flags() {
}

func findRelevantImages(client *lxd.Client) ([]imageDesc, error) {
	images, err := client.ListImages()
	if err != nil {
		return nil, err
	}

	imageDescs := make([]imageDesc, len(images))
	for idx, image := range images {
		if len(image.Aliases) == 0 {
			continue
		}

		alias := ""
		for _, tAl := range image.Aliases {
			if !strings.HasPrefix(tAl.Name, "ubuntu-sdk") {
				continue
			}

			slashIdx := strings.LastIndex(tAl.Name, "/")
			if (slashIdx <= 0){
				continue
			}
			alias = tAl.Name[0:slashIdx]
			break
		}

		if len(alias) == 0 {
			continue
		}

		imageDescs[idx].Alias = alias
		imageDescs[idx].Arch  = image.Architecture
		imageDescs[idx].Description = image.Properties["description"]
		imageDescs[idx].Fingerprint = image.Fingerprint
		imageDescs[idx].Size = image.Size
		imageDescs[idx].UploadDate = image.UploadDate
	}

	return imageDescs, nil
}

func (c *imagesCmd) run(args []string) error {

	config := ubuntu_sdk_tools.GetConfigOrDie()
	d, err := lxd.NewClient(config, "ubuntu-sdk-images")
	if err != nil {
		return err
	}

	imageDescs, err := findRelevantImages(d)
	if err != nil {
		return err
	}

	js, err := json.Marshal(imageDescs)
	if err != nil {
		return fmt.Errorf("Error while formatting data from the server. error: %v,\n", err)
	}
	fmt.Printf("%s\n", js)
	return nil
}
