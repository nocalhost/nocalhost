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

	funcs := []func() error{func() error { return testcase.PortForwardStart(cli, module, port) }}
	util.Retry("PortForward", funcs)

	clientgoutils.Must(testcase.PortForwardCheck(port))
	funcs = []func() error{func() error { return testcase.StatusCheckPortForward(cli, module, port) }}
	util.Retry("PortForward", funcs)

	funcs = []func() error{func() error { return testcase.PortForwardEnd(cli, module, port) }}
	util.Retry("PortForward", funcs)
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
	funcs := []func() error{
		func() error { return testcase.DevStart(cli, module) },
		func() error { return testcase.Sync(cli, module) },
		func() error { return testcase.DevEnd(cli, module) },
	}
	util.Retry("Dev", funcs)
}

func Sync(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	funcs := []func() error{
		func() error { return testcase.DevStart(cli, module) },
		func() error { return testcase.Sync(cli, module) },
		func() error { return testcase.SyncCheck(cli, module) },
		func() error { return testcase.SyncStatus(cli, module) },
	}
	util.Retry("Sync", funcs)
	_ = testcase.DevEnd(cli, module)
}

func StatefulSet(cli *nhctlcli.CLI, _ ...string) {
	module := "web"
	moduleType := "statefulset"
	funcs := []func() error{
		func() error { return testcase.DevStartT(cli, module, moduleType) },
		func() error { return testcase.SyncT(cli, module, moduleType) },
		func() error { return testcase.SyncCheckT(cli, module, moduleType) },
		func() error { return testcase.DevEndT(cli, module, moduleType) },
	}
	util.Retry("StatefulSet", funcs)
}

func Compatible(cli *nhctlcli.CLI, p ...string) {
	module := "ratings"
	port := 49080
	suiteName := "Compatible"
	util.Retry(suiteName, []func() error{func() error { return testcase.Exec(cli) }})
	m := []func() error{
		func() error { return testcase.DevStart(cli, module) },
		func() error { return testcase.Sync(cli, module) },
	}
	util.Retry(suiteName, m)
	m2 := []func() error{func() error { return testcase.PortForwardStart(cli, module, port) }}
	util.Retry(suiteName, m2)
	// install new version of nhctl
	if len(p) > 0 && p[0] != "" {
		util.Retry(suiteName, []func() error{func() error { return testcase.InstallNhctl(p[0]) }})
		//_ = testcase.RestartDaemon(cli)
		_ = testcase.NhctlVersion(cli)
	}
	funcsList := []func() error{
		func() error { return testcase.StatusCheck(cli, module) },
		func() error { return testcase.SyncCheck(cli, module) },
	}
	util.Retry(suiteName, funcsList)
	util.Retry(suiteName, []func() error{func() error { return testcase.PortForwardEnd(cli, module, port) }})
	//util.RetryWith2Params(suiteName,
	//	map[string]func(*nhctlcli.CLI, string) error{"DevEnd": testcase.DevEnd},
	//	cli,
	//	module)
	clientgoutils.Must(testcase.DevEnd(cli, module))
	// for temporary
	funcs := []func() error{
		func() error { return testcase.Upgrade(cli) },
		func() error { return testcase.Config(cli) },
		func() error { return testcase.List(cli) },
		func() error { return testcase.Db(cli) },
		func() error { return testcase.Pvc(cli) },
		func() error { return testcase.Reset(cli) },
		func() error { return testcase.InstallBookInfoThreeTimes(cli) },
	}
	util.Retry(suiteName, funcs)
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
	util.Retry("Apply", []func() error{func() error { return testcase.Apply(cli) }})
	clientgoutils.Must(testcase.List(cli))
}

func Upgrade(cli *nhctlcli.CLI, _ ...string) {
	util.Retry("Upgrade", []func() error{func() error { return testcase.Upgrade(cli) }})
	clientgoutils.Must(testcase.List(cli))
}

func Profile(cli *nhctlcli.CLI, _ ...string) {
	util.Retry("Profile", []func() error{
		func() error { return testcase.ProfileGetUbuntuWithJson(cli) },
		func() error { return testcase.ProfileGetDetailsWithoutJson(cli) },
		func() error { return testcase.ProfileSetDetails(cli) },
	})
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
	util.Retry("Prepare", []func() error{func() error { return testcase.InstallNhctl(v1) }})
	kubeconfig := util.GetKubeconfig()
	nocalhost := "nocalhost"
	tempCli := nhctlcli.NewNhctl(nocalhost, kubeconfig)
	clientgoutils.Must(testcase.NhctlVersion(tempCli))
	_ = testcase.StopDaemon(tempCli)
	util.Retry("Prepare", []func() error{func() error { return testcase.Init(tempCli) }})
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
