/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package suite

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/homedir"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"nocalhost/test/cluster"
	"nocalhost/test/runner"
	"nocalhost/test/testcase"
	"nocalhost/test/testdata"
	"nocalhost/test/util"
	"strings"
	"time"
)

func Hook(client runner.Client) {
	util.Retry(
		"Hook", []func() error{
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoHelmForTestHook(client)
					}, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "HelmHook")
					},
				)
			},
			func() error { return testcase.UpgradeBookInfoHelmForTestHook(client) },
			func() error { return testcase.UninstallBookInfoHelmForTestHook(client) },
		},
	)
}

func HelmAdaption(client runner.Client) {
	util.Retry(
		"HelmAdaption", []func() error{
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoUseHelmVals(client, "test-case", "bookinfohelm")
					}, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelm")
					},
				)
			},
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelm")
					}, nil,
				)
			},

			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoWithNativeHelm(client, "bookinfohelmnative")
					}, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelmnative")
					},
				)
			},
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelmnative")
					}, nil,
				)
			},

			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoWithNhctl(client, "bookinfohelmnhctl")
					}, func() error {
						return testcase.UninstallBookInfoWithNhctl(client, "bookinfohelmnhctl")
					},
				)
			},
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.UninstallBookInfoWithNhctl(client, "bookinfohelmnhctl")
					}, nil,
				)
			},

			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoWithNativeHelm(client, "bookinfohelmnativeother")
					}, func() error {
						return testcase.UninstallBookInfoWithNhctl(client, "bookinfohelmnativeother")
					},
				)
			},
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.UninstallBookInfoWithNhctl(client, "bookinfohelmnativeother")
					}, nil,
				)
			},

			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.InstallBookInfoWithNhctl(client, "bookinfohelmnativenhctlother")
					}, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelmnativenhctlother")
					},
				)
			},
			func() error {
				return util.TimeoutFunc(
					time.Minute*2, func() error {
						return testcase.UninstallBookInfoWithNativeHelm(client, "bookinfohelmnativenhctlother")
					}, nil,
				)
			},
		},
	)
}

func PortForward(client runner.Client, module, moduleType string) {
	port, err := ports.GetAvailablePort()
	if err != nil {
		port = 49088
	}

	util.Retry(
		fmt.Sprintf("PortForward-%s-%s", moduleType, module), []func() error{
			func() error { return testcase.PortForwardStartT(client, module, moduleType, port) },
		},
	)
	funcs := []func() error{
		func() error { return testcase.PortForwardCheck(port) },
		func() error { return testcase.StatusCheckPortForward(client, module, moduleType, port) },
		func() error { return testcase.PortForwardEndT(client, module, moduleType, port) },
	}
	util.Retry(fmt.Sprintf("PortForward-%s-%s", moduleType, module), funcs)
}

func PortForwardService(client runner.Client) error {
	module := "productpage"
	remotePort := 9080
	localPort, err := ports.GetAvailablePort()
	if err != nil {
		return errors.Errorf("fail to get available port, err: %s", err)
	}
	cmd := client.GetKubectl().Command(
		context.Background(),
		"port-forward",
		"service/"+module,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	)
	cmd.Stdout = log.TestLogger(client.SuiteName())
	cmd.Stderr = log.TestLogger(client.SuiteName())
	log.Infof("Running command: %v", cmd.Args)
	if err = cmd.Start(); err != nil {
		return errors.Errorf("fail to port-forward expose service-%s, err: %s", module, err)
	}
	defer cmd.Process.Kill()
	err = testcase.PortForwardCheck(localPort)
	return err
}

func test(cli runner.Client, moduleName, moduleType string, modeType profile.DevModeType) {
	PortForward(cli, moduleName, moduleType)
	funcs := []func() error{
		func() error { return PortForwardService(cli) },
		func() error {
			if err := testcase.DevStartT(cli, moduleName, moduleType, modeType); err != nil {
				_ = testcase.DevEndT(cli, moduleName, moduleType)
				return err
			}
			return nil
		},
		func() error { return testcase.SyncCheckT(cli, moduleName, moduleType) },
		func() error { return testcase.SyncStatusT(cli, moduleName, moduleType) },
		func() error { return testcase.DevEndT(cli, moduleName, moduleType) },
	}
	util.Retry(fmt.Sprintf("Dev-%s-%s-%s", modeType, moduleName, moduleType), funcs)
}

func Deployment(cli runner.Client) {
	test(cli, "ratings", "deployment", profile.ReplaceDevMode)
}

func DeploymentDuplicate(cli runner.Client) {
	test(cli, "ratings", "deployment", profile.DuplicateDevMode)
}

func StatefulSet(cli runner.Client) {
	test(cli, "web", "statefulset", profile.ReplaceDevMode)
}

func StatefulSetDuplicate(cli runner.Client) {
	test(cli, "web", "statefulset", profile.DuplicateDevMode)
}

/**
main step:
install a old version of nhctl
  (1) enter dev mode
  (2) start port-forward
  (3) start file sync
then, install a new version of nhctl
  (1) check sync status, developing status, port-forward status
  (2) check sync is ok or not
  (3) try to end port-forward
  (4) try to end dev mode
using new version of nhctl to do more operation
*/
func Compatible(cli runner.Client) {
	module := "ratings"
	port, err := ports.GetAvailablePort()
	if err != nil {
		port = 49080
	}
	suiteName := "Compatible"
	lastVersion, currentVersion := testcase.GetVersion()
	if lastVersion != "" {
		util.Retry(suiteName, []func() error{func() error { return testcase.InstallNhctl(lastVersion) }})
		util.Retry(suiteName, []func() error{func() error { return testcase.StopDaemon(cli.GetNhctl()) }})
		_ = testcase.NhctlVersion(cli.GetNhctl())
	}
	util.Retry(suiteName, []func() error{func() error { return testcase.Exec(cli) }})
	m := []func() error{
		func() error { return testcase.DevStartDeployment(cli, module) },
		func() error { return testcase.Sync(cli, module) },
	}
	util.Retry(suiteName, m)
	m2 := []func() error{func() error { return testcase.PortForwardStart(cli, module, port) }}
	util.Retry(suiteName, m2)
	// install new version of nhctl
	if lastVersion != "" {
		util.Retry(suiteName, []func() error{func() error { return testcase.InstallNhctl(currentVersion) }})
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
	clientgoutils.Must(testcase.DevEndDeployment(cli, module))
	// for temporary
	funcs := []func() error{
		func() error { return testcase.Upgrade(cli) },
		func() error { return testcase.Config(cli) },
		func() error { return testcase.List(cli) },
		//func() error { return testcase.Db(cli) },
		func() error { return testcase.Pvc(cli) },
		func() error { return testcase.InstallBookInfoDifferentType(cli) },
	}
	util.Retry(suiteName, funcs)
}

func Reset(cli runner.Client) {
	_ = testcase.UninstallBookInfo(cli)
	retryTimes := 5
	var err error
	for i := 0; i < retryTimes; i++ {
		timeoutCtx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
		if err = testcase.InstallBookInfo(timeoutCtx, cli); err != nil {
			log.Infof("install bookinfo error, error: %v, retrying...", err)
			_ = testcase.UninstallBookInfo(cli)
			continue
		}
		break
	}
	clientgoutils.Must(err)
	clientgoutils.Must(testcase.List(cli))
}

func Apply(cli runner.Client) {
	util.Retry("Apply", []func() error{func() error { return testcase.Apply(cli) }})
	clientgoutils.Must(testcase.List(cli))
}

func Upgrade(cli runner.Client) {
	util.Retry("Upgrade", []func() error{func() error { return testcase.Upgrade(cli) }})
	clientgoutils.Must(testcase.List(cli))
	Reset(cli)
	Apply(cli)
}

func ProfileAndAssociate(cli runner.Client) {

	singleSvcConfig := fp.NewRandomTempPath()
	multiSvcConfig := fp.NewRandomTempPath()
	fullConfig := fp.NewRandomTempPath()

	singleSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	multiSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	fullConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")

	util.Retry(
		"ProfileAndAssociate", []func() error{

			// clear env

			// 0
			func() error {
				_, _, _ = cli.GetKubectl().Run(context.TODO(), "delete", "configmap", "dev.nocalhost.config.bookinfo")
				return nil
			},
			// 1
			func() error { return testcase.DeAssociate(cli, "details", "deployment") },
			// 2
			func() error { return testcase.DeAssociate(cli, "ratings", "deployment") },

			// 3
			func() error {
				return singleSvcConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.SingleSvcConfig)
			},
			// 4
			func() error {
				return multiSvcConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.MultipleSvcConfig)
			},
			// 5
			func() error {
				return fullConfig.
					RelOrAbs(".nocalhost").
					MkdirThen().
					RelOrAbs("config.yaml").
					WriteFile(testdata.FullConfig)
			},
			// 6
			func() error {
				return singleSvcConfigCm.
					WriteFile(testdata.SingleSvcConfigCm)
			},
			// 7
			func() error {
				return multiSvcConfigCm.
					WriteFile(testdata.MultipleSvcConfigCm)
			},
			// 8
			func() error {
				return fullConfigCm.
					WriteFile(testdata.FullConfigCm)
			},
			// 9
			func() error { return testcase.ProfileGetUbuntuWithJson(cli) },
			// 10
			func() error { return testcase.ProfileGetDetailsWithoutJson(cli) },
			// 11
			func() error { return testcase.ProfileSetDetails(cli) },

			// test cfg load from cm
			// 12
			func() error { return testcase.ApplyCmForConfig(cli, singleSvcConfigCm) },
			// 13
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "singleSvcConfigCm") },

			// 14
			func() error { return testcase.ApplyCmForConfig(cli, multiSvcConfigCm) },
			// 15
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "multipleSvcConfig1Cm") },
			// 16
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "multipleSvcConfig2Cm") },

			// 17
			func() error { return testcase.ApplyCmForConfig(cli, fullConfigCm) },
			// 18
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			// 19
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },

			// test cfg load from local
			// 20
			func() error { return testcase.Associate(cli, "details", "deployment", singleSvcConfig) },
			// 21
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "singleSvcConfig") },
			// 22
			func() error { return testcase.Associate(cli, "details", "deployment", multiSvcConfig) },
			// 23
			func() error { return testcase.Associate(cli, "ratings", "deployment", multiSvcConfig) },
			// 24
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "multipleSvcConfig1") },
			// 25
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "multipleSvcConfig2") },
			// 26
			func() error { return testcase.Associate(cli, "details", "deployment", fullConfig) },
			// 27
			func() error { return testcase.Associate(cli, "ratings", "deployment", fullConfig) },
			// 28
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1") },
			// 29
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2") },

			// de associate the cfg, then will load from local
			// 30
			func() error { return testcase.DeAssociate(cli, "details", "deployment") },
			// 31
			func() error { return testcase.DeAssociate(cli, "ratings", "deployment") },

			// 32
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			// 33
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },

			// clean env
			// 34
			func() error {
				_, _, _ = cli.GetKubectl().Run(context.TODO(), "delete", "configmap", "dev.nocalhost.config.bookinfo")
				return nil
			},

			// config will not change, after env clean and no local, cm, annotation cfg
			// 35
			func() error { return testcase.ValidateImage(cli, "details", "deployment", "fullConfig1Cm") },
			// 36
			func() error { return testcase.ValidateImage(cli, "ratings", "deployment", "fullConfig2Cm") },
			// 37
			func() error { return testcase.ConfigReload(cli) },
		},
	)
	clientgoutils.Must(testcase.List(cli))
}

func Install(cli runner.Client) {
	retryTimes := 5
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfoDifferentType(cli); err != nil {
			log.Info(err)
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
	t := cluster.NewNothing()
	kubeconfig, err := t.Create()
	if err != nil {
		log.Info(err)
		if t != nil {
			t.Delete()
		}
		panic(err)
	}
	cancelFunc = func() {
		LogsForArchive()
		if errs := recover(); errs != nil {
			log.Infof("ignores timeout archive panic %v", errs)
		}
		t.Delete()
	}
	defer func() {
		if errs := recover(); errs != nil {
			LogsForArchive()
			t.Delete()
			panic(errs)
		}
	}()

	go util.TimeoutChecker(30*time.Minute, cancelFunc)

	_, currentVersion := testcase.GetVersion()
	util.Retry("Prepare", []func() error{func() error { return testcase.InstallNhctl(currentVersion) }})
	namespace := "test"
	tempCli := runner.NewNhctl(namespace, kubeconfig, "Prepare")
	clientgoutils.Must(testcase.NhctlVersion(tempCli))
	_ = testcase.StopDaemon(tempCli)

	webAddr := ""
	for i := 2; i >= 0; i-- {
		addr, err := testcase.Init(tempCli)
		if err == nil {
			webAddr = addr
			break
		} else if i == 0 {
			clientgoutils.Must(err)
		}
	}

	kubeconfigResult, err = testcase.GetKubeconfig(webAddr, namespace, kubeconfig)
	clientgoutils.Must(err)
	namespaceResult, err = clientgoutils.GetNamespaceFromKubeConfig(kubeconfigResult)
	clientgoutils.Must(err)
	return
}

func KillSyncthingProcess(cli runner.Client) {
	module := "ratings"
	funcs := []func() error{
		func() error {
			if err := testcase.DevStartDeployment(cli, module); err != nil {
				_ = testcase.DevEndDeployment(cli, module)
				return err
			}
			return nil
		},
		func() error { return testcase.SyncCheck(cli, module) },
		func() error { return testcase.SyncStatus(cli, module) },
		func() error { utils2.KillSyncthingProcess(cli.GetKubectl().Namespace); return nil },
		func() error { time.Sleep(time.Second * 2); return nil },
		func() error { return testcase.SyncCheck(cli, module) },
		func() error { return testcase.SyncStatus(cli, module) },
		func() error { return testcase.DevEndDeployment(cli, module) },
	}
	util.Retry("kill syncthing process", funcs)
}

func Get(cli runner.Client) {
	appName := "bookinfo-test"
	// kubectl annotate --overwrite deployments productpage dev.nocalhost/application-name=bookinfo111
	_, _, _ = cli.GetKubectl().RunWithRollingOut(context.Background(),
		"annotate",
		"deployments/productpage",
		fmt.Sprintf("%s=%s", _const.NocalhostApplicationName, appName),
		"--overwrite")
	// wait for informer to parse app from annotation
	<-time.Tick(time.Second * 10)
	cases := []struct {
		resource string
		appName  string
		keywords []string
	}{
		{resource: "deployments", appName: "bookinfo", keywords: []string{"details", "ratings", "reviews"}},
		{resource: "jobs", appName: "bookinfo", keywords: []string{"print-num-01"}},
		{resource: "service", appName: "bookinfo", keywords: []string{"details", "ratings", "reviews"}},
		{resource: "pods", appName: "", keywords: []string{"details", "ratings", "reviews"}},
		{resource: "app", appName: "", keywords: []string{"bookinfo", appName}},
	}

	funcs := []func() error{
		func() error {
			for _, item := range cases {
				err := testcase.Get(
					cli, item.resource, item.appName, func(result string) error {
						for _, s := range item.keywords {
							if !strings.Contains(result, s) {
								return errors.Errorf("nhctl get %s, result not contains resource: %s", item.resource, s)
							}
						}
						return nil
					},
				)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	util.Retry("get", funcs)
}

func TestLog(_ runner.Client) {
	file := fp.NewFilePath(homedir.HomeDir()).
		RelOrAbs(".nh").
		RelOrAbs("nhctl").
		RelOrAbs("logs").
		RelOrAbs("nhctl.log").
		ReadFile()
	if len(file) == 0 {
		panic("Daemon log file is empty, please check your log initialize code !!!")
	}
}
