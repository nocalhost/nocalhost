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
	"errors"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/tke"
	"nocalhost/test/util"
	"time"
)

func PortForward(cli *nhctlcli.CLI, _ ...string) {
	module := "reviews"
	port := 49080
	testcase.PortForwardStart(cli, module, port)
	testcase.PortForwardCheck(port)
	testcase.StatusCheckPortForward(cli, module, port)
	testcase.PortForwardEnd(cli, module, port)
}

func Dev(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	testcase.DevStart(cli, module)
	testcase.DevEnd(cli, module)
}

func Sync(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	testcase.DevStart(cli, module)
	testcase.Sync(cli, module)
	testcase.SyncStatus(cli, module)
	testcase.SyncCheck(cli, module)
	testcase.DevEnd(cli, module)
}

func Compatible(cli *nhctlcli.CLI, p ...string) {
	module := "ratings"
	port := 49080
	testcase.DevStart(cli, module)
	testcase.Sync(cli, module)
	testcase.PortForwardStart(cli, module, port)
	if len(p) > 0 && p[0] != "" {
		testcase.InstallNhctl(p[0])
		testcase.RestartDaemon(cli)
		testcase.NhctlVersion(cli)
	}
	testcase.StatusCheck(cli, module)
	testcase.SyncCheck(cli, module)
	testcase.PortForwardEnd(cli, module, port)
	testcase.DevEnd(cli, module)
	// for temporary
	testcase.Upgrade(cli)
	testcase.Config(cli)
	testcase.List(cli)
	testcase.Db(cli)
	testcase.Pvc(cli)
	testcase.Reset(cli)
	testcase.InstallBookInfoThreeTimes(cli)
	testcase.Exec(cli)
}

func Reset(cli *nhctlcli.CLI, _ ...string) {
	testcase.Reset(cli)
	testcase.InstallBookInfo(cli)
}

func Apply(cli *nhctlcli.CLI, _ ...string) {
	testcase.Apply(cli)
}

func Upgrade(cli *nhctlcli.CLI, _ ...string) {
	testcase.InstallBookInfo(cli)
	testcase.Upgrade(cli)
}

func Install(cli *nhctlcli.CLI, _ ...string) {
	testcase.InstallBookInfoThreeTimes(cli)
	//testcase.PortForwardCheck(39080)
}

// Prepare will install a nhctl client, create a k8s cluster if necessary
func Prepare() (cli *nhctlcli.CLI, v1 string, v2 string, cancelFunc func()) {
	if util.NeedsToInitK8sOnTke() {
		t := tke.CreateK8s()
		cancelFunc = t.Delete
		defer func() {
			if err := recover(); err != nil {
				t.Delete()
				panic(err)
			}
		}()
	}
	go util.TimeoutChecker(1*time.Hour, cancelFunc)
	v1, v2 = testcase.GetVersion()
	testcase.InstallNhctl(v1)
	kubeconfig := util.GetKubeconfig()

	tempCli := nhctlcli.NewNhctl("", kubeconfig)
	util.Init(tempCli)
	testcase.NhctlVersion(tempCli)
	testcase.StopDaemon(tempCli)
	go testcase.Init(tempCli)
	if i := <-testcase.StatusChan; i != 0 {
		panic("Init nocalhost occurs error, exiting")
	}
	web := <-testcase.WebServerEndpointChan
	var ns string
	var err error
	newKubeconfig := testcase.GetKubeconfig(ns, web, kubeconfig)
	if ns, err = clientgoutils.GetNamespaceFromKubeConfig(newKubeconfig); err != nil {
		panic(err)
	}
	if ns == "" {
		panic(errors.New("--namespace or --kubeconfig mush be provided"))
	}
	cli = nhctlcli.NewNhctl(ns, newKubeconfig)
	return
}
