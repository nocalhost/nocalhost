/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nocalhost

import (
	"fmt"
	"testing"

	"nocalhost/cmd/nhctl/cmds"
	"nocalhost/pkg/nhctl/utils"
)

func TestNocalHost_GetApplicationDir(t *testing.T) {
	//n := NocalHost{}
	//fmt.Println(n.GetApplicationDir())
}

func TestNocalHost_GetApplications(t *testing.T) {
	n := NocalHost{}
	apps, err := n.GetApplicationNames()
	utils.Mush(err)
	for _, app := range apps {
		fmt.Println(app)
	}
}

func TestListApplications(t *testing.T) {
	cmds.ListApplications()
}
