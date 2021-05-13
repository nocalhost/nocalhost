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
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/tke"
	"nocalhost/test/util"
	"strconv"
	"time"
)

func PortForward(cli *nhctlcli.CLI, _ ...string) {
	module := "reviews"
	port := 49080

	funcs := []func(*nhctlcli.CLI, string, int) error{testcase.PortForwardStart}
	util.RetryWith3Params("PortForward", funcs, cli, module, port)

	clientgoutils.Must(testcase.PortForwardCheck(port))
	funcs = []func(*nhctlcli.CLI, string, int) error{testcase.StatusCheckPortForward}
	util.RetryWith3Params("PortForward", funcs, cli, module, port)

	funcs = []func(*nhctlcli.CLI, string, int) error{testcase.PortForwardEnd}
	util.RetryWith3Params("PortForward", funcs, cli, module, port)
}

func PortForwardService(cli *nhctlcli.CLI, _ ...string) {
	module := "productpage"
	removePort := 9080
	localPort, err := ports.GetAvailablePort()
	if err != nil {
		panic(errors.Errorf("fail to get available port, err: %s", err))
	}
	kubectl := nhctlcli.NewKubectl(cli.Namespace, cli.KubeConfig)
	cmd := kubectl.Command(context.Background(),
		"port-forward",
		"service/"+module,
		strconv.Itoa(localPort)+":"+strconv.Itoa(removePort),
	)
	log.Infof("Running command: %v", cmd.Args)
	if err = cmd.Start(); err != nil {
		panic(errors.Errorf("fail to port-forward expose service-%s, err: %s", module, err))
	}
	clientgoutils.Must(testcase.PortForwardCheck(localPort))
	_ = cmd.Process.Kill()
}

func Dev(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	funcs := []func(*nhctlcli.CLI, string) error{testcase.DevStart, testcase.Sync, testcase.DevEnd}
	util.RetryWith2Params("Dev", funcs, cli, module)
}

func Sync(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	funcs := []func(*nhctlcli.CLI, string) error{testcase.DevStart, testcase.Sync, testcase.SyncCheck, testcase.SyncStatus}
	util.RetryWith2Params("Sync", funcs, cli, module)
	_ = testcase.DevEnd(cli, module)
}

func Compatible(cli *nhctlcli.CLI, p ...string) {
	module := "ratings"
	port := 49080
	suiteName := "Compatible"
	util.RetryWith1Params(suiteName, []func(*nhctlcli.CLI) error{testcase.Exec}, cli)
	m := []func(*nhctlcli.CLI, string) error{testcase.DevStart, testcase.Sync}
	util.RetryWith2Params(suiteName, m, cli, module)
	m2 := []func(*nhctlcli.CLI, string, int) error{testcase.PortForwardStart}
	util.RetryWith3Params(suiteName, m2, cli, module, port)
	// install new version of nhctl
	if len(p) > 0 && p[0] != "" {
		util.RetryWithString(suiteName, []func(string) error{testcase.InstallNhctl}, p[0])
		_ = testcase.RestartDaemon(cli)
		_ = testcase.NhctlVersion(cli)
	}
	funcsList := []func(*nhctlcli.CLI, string) error{testcase.StatusCheck, testcase.SyncCheck}
	util.RetryWith2Params(suiteName, funcsList, cli, module)
	util.RetryWith3Params(suiteName, []func(*nhctlcli.CLI, string, int) error{testcase.PortForwardEnd},
		cli, module, port)
	//util.RetryWith2Params(suiteName,
	//	map[string]func(*nhctlcli.CLI, string) error{"DevEnd": testcase.DevEnd},
	//	cli,
	//	module)
	clientgoutils.Must(testcase.DevEnd(cli, module))
	// for temporary
	funcs := []func(*nhctlcli.CLI) error{
		testcase.Upgrade,
		testcase.Config,
		testcase.List,
		testcase.Db,
		testcase.Pvc,
		testcase.Reset,
		testcase.InstallBookInfoThreeTimes,
	}
	util.RetryWith1Params(suiteName, funcs, cli)
}

func Reset(cli *nhctlcli.CLI, _ ...string) {
	clientgoutils.Must(testcase.Reset(cli))
	_ = testcase.UninstallBookInfo(cli)
	retryTimes := 5
	var err error
	clientgoutils.Must(err)
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfo(cli); err != nil {
			log.Infof("install bookinfo error, error: %v, retrying...", err)
			_ = testcase.UninstallBookInfo(cli)
			_ = testcase.Reset(cli)
			continue
		}
		break
	}
	clientgoutils.Must(err)
	clientgoutils.Must(testcase.List(cli))
}

func Apply(cli *nhctlcli.CLI, _ ...string) {
	util.RetryWith1Params("Apply", []func(*nhctlcli.CLI) error{testcase.Apply}, cli)
	clientgoutils.Must(testcase.List(cli))
}

func Upgrade(cli *nhctlcli.CLI, _ ...string) {
	util.RetryWith1Params("Upgrade", []func(*nhctlcli.CLI) error{testcase.Upgrade}, cli)
	clientgoutils.Must(testcase.List(cli))
}

func Install(cli *nhctlcli.CLI, _ ...string) {
	retryTimes := 5
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfoThreeTimes(cli); err != nil {
			log.Info(err)
			_ = testcase.Reset(cli)
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
			log.Info(err)
			if t != nil {
				t.Delete()
			}
			panic(err)
		}
		cancelFunc = t.Delete
		defer func() {
			if errs := recover(); errs != nil {
				t.Delete()
				panic(errs)
			}
		}()
	}
	go util.TimeoutChecker(1*time.Hour, cancelFunc)
	v1, v2 = testcase.GetVersion()
	util.RetryWithString("Prepare", []func(string) error{testcase.InstallNhctl}, v1)
	kubeconfig := util.GetKubeconfig()
	nocalhost := "nocalhost"
	tempCli := nhctlcli.NewNhctl(nocalhost, kubeconfig)
	clientgoutils.Must(testcase.NhctlVersion(tempCli))
	_ = testcase.StopDaemon(tempCli)
	util.RetryWith1Params("Prepare", []func(*nhctlcli.CLI) error{testcase.Init}, tempCli)
	newKubeconfig, err := testcase.GetKubeconfig(nocalhost, kubeconfig)
	clientgoutils.Must(err)
	ns, err := clientgoutils.GetNamespaceFromKubeConfig(newKubeconfig)
	clientgoutils.Must(err)
	if ns == "" {
		panic(errors.New("--namespace or --kubeconfig must be provided"))
	}
	cli = nhctlcli.NewNhctl(ns, newKubeconfig)
	clientgoutils.Must(util.Init(cli))
	return
}
