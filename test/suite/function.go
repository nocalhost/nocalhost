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
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/testcase"
	"nocalhost/test/testdata"
	"nocalhost/test/tke"
	"nocalhost/test/util"
	"time"
)

func HelmAdaption(client runner.Client, _ ...string) {
	util.Retry(
		"HelmAdaption", []func() error{
			func() error { return testcase.InstallBookInfoWithNativeHelm(client) },
			func() error { return testcase.UninstallBookInfoWithNativeHelm(client) },

			func() error { return testcase.InstallBookInfoWithNhctl(client) },
			func() error { return testcase.UninstallBookInfoWithNhctl(client) },

			func() error { return testcase.InstallBookInfoWithNativeHelm(client) },
			func() error { return testcase.UninstallBookInfoWithNhctl(client) },

			func() error { return testcase.InstallBookInfoWithNhctl(client) },
			func() error { return testcase.UninstallBookInfoWithNativeHelm(client) },
		},
	)
}

func PortForward(client runner.Client, _ ...string) {
	module := "reviews"
	port := 49080

	//funcs := []func() error{func() error { return testcase.PortForwardStart(cli, module, port) }}
	//util.Retry("PortForward", funcs)

	//clientgoutils.Must(testcase.PortForwardCheck(port))
	funcs := []func() error{
		func() error { return testcase.PortForwardStart(client, module, port) },
		func() error { return testcase.PortForwardCheck(port) },
		func() error { return testcase.StatusCheckPortForward(client, module, port) },
		func() error { return testcase.PortForwardEnd(client, module, port) },
	}
	util.Retry("PortForward", funcs)

	//funcs = []func() error{func() error { return testcase.PortForwardEnd(cli, module, port) }}
	//util.Retry("PortForward", funcs)
}

func PortForwardService(client runner.Client, _ ...string) {
	module := "productpage"
	remotePort := 9080
	localPort, err := ports.GetAvailablePort()
	if err != nil {
		panic(errors.Errorf("fail to get available port, err: %s", err))
	}
	cmd := client.GetKubectl().Command(
		context.Background(),
		"port-forward",
		"service/"+module,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	)
	log.Infof("Running command: %v", cmd.Args)
	if err = cmd.Start(); err != nil {
		panic(errors.Errorf("fail to port-forward expose service-%s, err: %s", module, err))
	}
	clientgoutils.Must(testcase.PortForwardCheck(localPort))
	_ = cmd.Process.Kill()
}

func Deployment(cli runner.Client, _ ...string) {
	PortForward(cli)
	PortForwardService(cli)
	module := "ratings"
	funcs := []func() error{

		func() error {
			if err := testcase.DevStart(cli, module); err != nil {
				_ = testcase.DevEnd(cli, module)
				return err
			}
			return nil
		},
		func() error { return testcase.Sync(cli, module) },
		func() error { return testcase.SyncCheck(cli, module) },
		func() error { return testcase.SyncStatus(cli, module) },
		func() error { return testcase.DevEnd(cli, module) },
	}
	util.Retry("Dev", funcs)
}

//func Sync(cli nhctlcli.Client, _ ...string) {
//	module := "ratings"
//	funcs := []func() error{
//		func() error { return testcase.DevStart(cli, module) },
//		func() error { return testcase.Sync(cli, module) },
//		func() error { return testcase.SyncCheck(cli, module) },
//		func() error { return testcase.SyncStatus(cli, module) },
//	}
//	util.Retry("Sync", funcs)
//	_ = testcase.DevEnd(cli, module)
//}

func StatefulSet(cli runner.Client, _ ...string) {
	module := "web"
	moduleType := "statefulset"
	funcs := []func() error{
		func() error { return testcase.DevStartT(cli, module, moduleType) },
		func() error { return testcase.SyncT(cli, module, moduleType) },
		func() error { return testcase.SyncCheckT(cli, cli.NameSpace(), module, moduleType) },
		func() error { return testcase.DevEndT(cli, module, moduleType) },
	}
	util.Retry("StatefulSet", funcs)
}

func Compatible(cli runner.Client, p ...string) {
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
		_ = testcase.NhctlVersion(cli.GetNhctl())
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

func Reset(cli runner.Client, _ ...string) {
	clientgoutils.Must(testcase.Reset(cli))
	_ = testcase.UninstallBookInfo(cli)
	retryTimes := 5
	var err error
	for i := 0; i < retryTimes; i++ {
		timeoutCtx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
		if err = testcase.InstallBookInfo(timeoutCtx, cli); err != nil {
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

func Apply(cli runner.Client, _ ...string) {
	util.Retry("Apply", []func() error{func() error { return testcase.Apply(cli) }})
	clientgoutils.Must(testcase.List(cli))
}

func Upgrade(cli runner.Client, _ ...string) {
	util.Retry("Upgrade", []func() error{func() error { return testcase.Upgrade(cli) }})
	clientgoutils.Must(testcase.List(cli))
	Reset(cli)
	Apply(cli)
	Profile(cli)
}

func Profile(cli runner.Client, _ ...string) {

	singleSvcConfig := fp.NewRandomTempPath()
	multiSvcConfig := fp.NewRandomTempPath()
	fullConfig := fp.NewRandomTempPath()

	singleSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	multiSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	fullConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")

	util.Retry(
		"Profile", []func() error{

			// clear env
			func() error {
				_, _, _ = cli.GetKubectl().Run(context.TODO(), "delete", "configmap", "dev.nocalhost.config.bookinfo")
				return nil
			},
			func() error { return testcase.DeAssociate(cli, "details", "deployment") },
			func() error { return testcase.DeAssociate(cli, "ratings", "deployment") },

			func() error {
				return singleSvcConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.SingleSvcConfig)
			},
			func() error {
				return multiSvcConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.MultipleSvcConfig)
			},
			func() error {
				return fullConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.FullConfig)
			},

			func() error {
				return singleSvcConfigCm.
					WriteFile(testdata.SingleSvcConfigCm)
			},
			func() error {
				return multiSvcConfigCm.
					WriteFile(testdata.MultipleSvcConfigCm)
			},
			func() error {
				return fullConfigCm.
					WriteFile(testdata.FullConfigCm)
			},

			func() error { return testcase.ProfileGetUbuntuWithJson(cli) },
			func() error { return testcase.ProfileGetDetailsWithoutJson(cli) },
			func() error { return testcase.ProfileSetDetails(cli) },

			// test cfg load from cm
			func() error { return testcase.ApplyCmForConfig(cli, singleSvcConfigCm) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "singleSvcConfigCm") },

			func() error { return testcase.ApplyCmForConfig(cli, multiSvcConfigCm) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "multipleSvcConfig1Cm") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "multipleSvcConfig2Cm") },

			func() error { return testcase.ApplyCmForConfig(cli, fullConfigCm) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },

			// test cfg load from local
			func() error { return testcase.Associate(cli, "details", "deployment", singleSvcConfig) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "singleSvcConfig") },

			func() error { return testcase.Associate(cli, "details", "deployment", multiSvcConfig) },
			func() error { return testcase.Associate(cli, "ratings", "deployment", multiSvcConfig) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "multipleSvcConfig1") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "multipleSvcConfig2") },

			func() error { return testcase.Associate(cli, "details", "deployment", fullConfig) },
			func() error { return testcase.Associate(cli, "ratings", "deployment", fullConfig) },
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2") },

			// de associate the cfg, then will load from local
			func() error { return testcase.DeAssociate(cli, "details", "deployment") },
			func() error { return testcase.DeAssociate(cli, "ratings", "deployment") },

			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },

			// clean env
			func() error {
				_, _, _ = cli.GetKubectl().Run(context.TODO(), "delete", "configmap", "dev.nocalhost.config.bookinfo")
				return nil
			},

			// config will not change, after env clean and no local, cm, annotation cfg
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },

			func() error { return testcase.ConfigReload(cli) },
		},
	)
	clientgoutils.Must(testcase.List(cli))
}

func Install(cli runner.Client, _ ...string) {
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
func Prepare() (cancelFunc func(), namespaceResult, kubeconfigResult string) {
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
	v1, _ := testcase.GetVersion()
	util.Retry("Prepare", []func() error{func() error { return testcase.InstallNhctl(v1) }})
	kubeconfig := util.GetKubeconfig()
	nocalhost := "nocalhost"
	tempCli := runner.NewNhctl(nocalhost, kubeconfig)
	clientgoutils.Must(testcase.NhctlVersion(tempCli))
	_ = testcase.StopDaemon(tempCli)
	util.Retry("Prepare", []func() error{func() error { return testcase.Init(tempCli) }})

	kubeconfigResult, err := testcase.GetKubeconfig(nocalhost, kubeconfig)
	clientgoutils.Must(err)
	namespaceResult, err = clientgoutils.GetNamespaceFromKubeConfig(kubeconfigResult)
	clientgoutils.Must(err)
	return
}
