/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package suite

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/util"
)

// test suite
type T struct {
	Cli       *nhctlcli.CLI
	CleanFunc func()
}

// Run command and clean environment after finished
func (t *T) Run(name string, fn func(cli *nhctlcli.CLI, p ...string), pp ...string) {
	testcase.InstallBookInfo(t.Cli)
	util.WaitToBeStatus(t.Cli.Namespace, "pods", "app=reviews", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	util.WaitToBeStatus(t.Cli.Namespace, "pods", "app=ratings", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	fmt.Println("Testing " + name)
	defer func() {
		if err := recover(); err != nil {
			t.Clean()
			panic(err)
		}
	}()
	fn(t.Cli, pp...)
	fmt.Println("Testing done " + name)
	//testcase.Reset(t.Cli)
	testcase.UninstallBookInfo(t.Cli)
}

func (t *T) Clean() {
	if t.CleanFunc != nil {
		t.CleanFunc()
	}
}
