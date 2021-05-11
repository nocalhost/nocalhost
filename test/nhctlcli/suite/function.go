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
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/tke"
	"nocalhost/test/util"
	"time"
)

func PortForward(cli *nhctlcli.CLI, _ ...string) {
	module := "reviews"
	port := 49080

	funcs := map[string]func(*nhctlcli.CLI, string, int) error{"PortForwardStart": testcase.PortForwardStart}
	util.RetryWith3Params(funcs, "PortForward", cli, module, port)
	clientgoutils.Must(testcase.PortForwardCheck(port))
	funcs = map[string]func(*nhctlcli.CLI, string, int) error{"StatusCheckPortForward": testcase.StatusCheckPortForward}
	util.RetryWith3Params(funcs, "PortForward", cli, module, port)
	funcs = map[string]func(*nhctlcli.CLI, string, int) error{"PortForwardEnd": testcase.PortForwardEnd}
	util.RetryWith3Params(funcs, "PortForward", cli, module, port)
}

func Dev(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	funcs := map[string]func(*nhctlcli.CLI, string) error{"DevStart": testcase.DevStart, "DevEnd": testcase.DevEnd}
	util.RetryWith2Params(funcs, "Dev", cli, module)
}

func Sync(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	funcs := map[string]func(*nhctlcli.CLI, string) error{
		"DevStart":   testcase.DevStart,
		"Sync":       testcase.Sync,
		"SyncStatus": testcase.SyncStatus,
		"DevEnd":     testcase.DevEnd,
	}
	util.RetryWith2Params(funcs, "sync", cli, module)
}

func Compatible(cli *nhctlcli.CLI, p ...string) {
	module := "ratings"
	port := 49080
	util.RetryWith1Params(map[string]func(*nhctlcli.CLI) error{"Exec": testcase.Exec}, "compatible", cli)
	m := map[string]func(*nhctlcli.CLI, string) error{"DevStart": testcase.DevStart, "Sync": testcase.Sync}
	util.RetryWith2Params(m, "compatible", cli, module)
	m2 := map[string]func(*nhctlcli.CLI, string, int) error{"PortForwardStart": testcase.PortForwardStart}
	util.RetryWith3Params(m2, "compatible", cli, module, port)
	if len(p) > 0 && p[0] != "" {
		util.RetryWithString(map[string]func(string) error{"InstallNhctl": testcase.InstallNhctl}, "compatible", p[0])
		_ = testcase.RestartDaemon(cli)
		_ = testcase.NhctlVersion(cli)
	}
	funcsList := map[string]func(*nhctlcli.CLI, string) error{
		"StatusCheck": testcase.StatusCheck,
		"SyncCheck":   testcase.SyncCheck,
	}
	util.RetryWith2Params(funcsList, "compatible", cli, module)
	util.RetryWith3Params(
		map[string]func(*nhctlcli.CLI, string, int) error{"PortForwardEnd": testcase.PortForwardEnd},
		"compatible", cli, module, port)
	util.RetryWith2Params(
		map[string]func(*nhctlcli.CLI, string) error{"DevEnd": testcase.DevEnd},
		"compatible",
		cli,
		module)
	// for temporary
	funcs := map[string]func(*nhctlcli.CLI) error{
		"Upgrade":                   testcase.Upgrade,
		"Config":                    testcase.Config,
		"List":                      testcase.List,
		"Db":                        testcase.Db,
		"Pvc":                       testcase.Pvc,
		"Reset":                     testcase.Reset,
		"InstallBookInfoThreeTimes": testcase.InstallBookInfoThreeTimes,
	}
	util.RetryWith1Params(funcs, "compatible", cli)
}

func Reset(cli *nhctlcli.CLI, _ ...string) {
	util.RetryWith1Params(map[string]func(*nhctlcli.CLI) error{"Reset": testcase.Reset}, "Reset", cli)

	retryTimes := 5
	var err error
	clientgoutils.Must(err)
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfo(cli); err != nil {
			_ = testcase.Reset(cli)
			time.Sleep(time.Second * 2)
			continue
		}
		break
	}
	clientgoutils.Must(err)
	clientgoutils.Must(testcase.List(cli))
}

func Apply(cli *nhctlcli.CLI, _ ...string) {
	util.RetryWith1Params(map[string]func(*nhctlcli.CLI) error{"Apply": testcase.Apply}, "Apply", cli)
	clientgoutils.Must(testcase.List(cli))
}

func Upgrade(cli *nhctlcli.CLI, _ ...string) {
	util.RetryWith1Params(map[string]func(*nhctlcli.CLI) error{"Upgrade": testcase.Upgrade}, "Upgrade", cli)
	clientgoutils.Must(testcase.List(cli))
}

func Install(cli *nhctlcli.CLI, _ ...string) {
	retryTimes := 5
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfoThreeTimes(cli); err != nil {
			_ = testcase.Reset(cli)
			time.Sleep(time.Second * 2)
			continue
		}
		break
	}
	if err != nil {
		panic(errors.New("test suite failed, fail on step: install"))
	}
}

// Prepare will install a nhctl client, create a k8s cluster if necessary
func Prepare() (cli *nhctlcli.CLI, v1 string, v2 string, cancelFunc func()) {
	if util.NeedsToInitK8sOnTke() {
		t, err := tke.CreateK8s()
		if err != nil {
			if t != nil {
				t.Delete()
			}
			panic(err)
		}
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
	var err error
	util.RetryWithString(
		map[string]func(string) error{"InstallNhctl": testcase.InstallNhctl}, "Prepare", v1)
	kubeconfig := util.GetKubeconfig()
	tempCli := nhctlcli.NewNhctl("", kubeconfig)
	clientgoutils.Must(util.Init(tempCli))
	clientgoutils.Must(testcase.NhctlVersion(tempCli))
	_ = testcase.StopDaemon(tempCli)
	util.RetryWith1Params(map[string]func(*nhctlcli.CLI) error{"Init": testcase.Init}, "Prepare", tempCli)
	log.Info("wait for api server endpoint")
	web := <-testcase.ApiServerEndpointChan
	var ns string
	newKubeconfig, err := testcase.GetKubeconfig(ns, web, kubeconfig)
	clientgoutils.Must(err)
	ns, err = clientgoutils.GetNamespaceFromKubeConfig(newKubeconfig)
	clientgoutils.Must(err)
	if ns == "" {
		panic(errors.New("--namespace or --kubeconfig must be provided"))
	}
	cli = nhctlcli.NewNhctl(ns, newKubeconfig)
	return
}
