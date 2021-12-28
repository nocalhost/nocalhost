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
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"nocalhost/test/runner"
	"nocalhost/test/testcase"
	"nocalhost/test/testdata"
	"nocalhost/test/tke"
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

func HelmAdaption(client runner.Client) error {
	return util.Retry(
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

func PortForward(client runner.Client, module, moduleType string) error {
	port, err := ports.GetAvailablePort()
	if err != nil {
		port = 49088
	}

	err = util.Retry(
		fmt.Sprintf("PortForward-%s-%s", moduleType, module), []func() error{
			func() error { return testcase.PortForwardStartT(client, module, moduleType, port) },
		},
	)
	if err != nil {
		return err
	}

	funcs := []func() error{
		func() error { return testcase.PortForwardCheck(port) },
		func() error { return testcase.StatusCheckPortForward(client, module, moduleType, port) },
		func() error { return testcase.PortForwardEndT(client, module, moduleType, port) },
	}
	return util.Retry(fmt.Sprintf("PortForward-%s-%s", moduleType, module), funcs)
}

func PortForwardService(client runner.Client) error {
	module := "productpage"
	remotePort := 9080
	localPort, err := ports.GetAvailablePort()
	if err != nil {
		//panic(errors.Errorf("fail to get available port, err: %s", err))
		return err
	}
	cmd := client.GetKubectl().Command(
		context.Background(),
		"port-forward",
		"service/"+module,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	)
	log.Infof("Running command: %v", cmd.Args)
	if err = cmd.Start(); err != nil {
		//panic(errors.Errorf("fail to port-forward expose service-%s, err: %s", module, err))
		return errors.WithStack(err)
	}
	//clientgoutils.Must(testcase.PortForwardCheck(localPort))
	err = testcase.PortForwardCheck(localPort)
	if err != nil {
		return err
	}
	_ = cmd.Process.Kill()
	return nil
}

func test(cli runner.Client, moduleName, moduleType string, modeType profile.DevModeType) error {
	err := PortForward(cli, moduleName, moduleType)
	if err != nil {
		return err
	}
	if err = PortForwardService(cli); err != nil {
		return err
	}
	funcs := []func() error{
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
	return util.Retry(fmt.Sprintf("Dev-%s-%s-%s", modeType, moduleName, moduleType), funcs)
}

func Deployment(cli runner.Client) error {
	return test(cli, "ratings", "deployment", profile.ReplaceDevMode)
}

func DeploymentDuplicate(cli runner.Client) error {
	return test(cli, "ratings", "deployment", profile.DuplicateDevMode)
}

func StatefulSet(cli runner.Client) error {
	return test(cli, "web", "statefulset", profile.ReplaceDevMode)
}

func StatefulSetDuplicate(cli runner.Client) error {
	return test(cli, "web", "statefulset", profile.DuplicateDevMode)
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
func Compatible(cli runner.Client) error {
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
	err = util.Retry(suiteName, []func() error{func() error { return testcase.Exec(cli) }})
	if err != nil {
		return err
	}
	m := []func() error{
		func() error { return testcase.DevStartDeployment(cli, module) },
		func() error { return testcase.Sync(cli, module) },
	}
	err = util.Retry(suiteName, m)
	if err != nil {
		return err
	}
	m2 := []func() error{func() error { return testcase.PortForwardStart(cli, module, port) }}
	err = util.Retry(suiteName, m2)
	if err != nil {
		return err
	}
	// install new version of nhctl
	if lastVersion != "" {
		util.Retry(suiteName, []func() error{func() error { return testcase.InstallNhctl(currentVersion) }})
		//_ = testcase.RestartDaemon(cli)
		_ = testcase.NhctlVersion(cli.GetNhctl())
	}
	funcsList := []func() error{
		func() error { return testcase.StatusCheck(cli, module) },
		func() error { return testcase.SyncCheck(cli, module) },
		func() error { return testcase.PortForwardEnd(cli, module, port) },
	}
	err = util.Retry(suiteName, funcsList)
	if err != nil {
		return err
	}

	//clientgoutils.Must(testcase.DevEndDeployment(cli, module))
	err = testcase.DevEndDeployment(cli, module)
	if err != nil {
		return err
	}
	// for temporary
	funcs := []func() error{
		func() error { return testcase.Upgrade(cli) },
		func() error { return testcase.Config(cli) },
		func() error { return testcase.List(cli) },
		func() error { return testcase.Pvc(cli) },
		func() error { return testcase.InstallBookInfoDifferentType(cli) },
	}
	return util.Retry(suiteName, funcs)
}

func Reset(cli runner.Client) error {
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
	//clientgoutils.Must(err)
	//clientgoutils.Must(testcase.List(cli))
	if err != nil {
		return err
	}
	return testcase.List(cli)
}

func Apply(cli runner.Client) error {
	err := util.Retry("Apply", []func() error{func() error { return testcase.Apply(cli) }})
	if err != nil {
		return err
	}
	//clientgoutils.Must(testcase.List(cli))
	return testcase.List(cli)
}

func Upgrade(cli runner.Client) error {
	err := util.Retry("Upgrade", []func() error{func() error { return testcase.Upgrade(cli) }})
	if err != nil {
		return err
	}

	//clientgoutils.Must(testcase.List(cli))
	if err = testcase.List(cli); err != nil {
		return err
	}

	if err = Reset(cli); err != nil {
		return err
	}

	return Apply(cli)
}

func ProfileAndAssociate(cli runner.Client) error {

	singleSvcConfig := fp.NewRandomTempPath()
	multiSvcConfig := fp.NewRandomTempPath()
	fullConfig := fp.NewRandomTempPath()

	singleSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	multiSvcConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")
	fullConfigCm := fp.NewRandomTempPath().MkdirThen().RelOrAbs("cm.yaml")

	err := util.Retry(
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
	if err != nil {
		return err
	}
	//clientgoutils.Must(testcase.List(cli))
	return testcase.List(cli)
}

func Install(cli runner.Client) error {
	retryTimes := 2
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfoDifferentType(cli); err != nil {
			log.Info(err)
			continue
		}
		break
	}
	//if err != nil {
	//	panic(errors.New("test suite failed, fail on step: install"))
	//}
	return err
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
	}

	go util.TimeoutChecker(1*time.Hour, cancelFunc)

	_, currentVersion := testcase.GetVersion()
	util.Retry("Prepare", []func() error{func() error { return testcase.InstallNhctl(currentVersion) }})
	kubeconfig := util.GetKubeconfig()
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

	kubeconfigResult, err := testcase.GetKubeconfig(webAddr, namespace, kubeconfig)
	clientgoutils.Must(err)
	namespaceResult, err = clientgoutils.GetNamespaceFromKubeConfig(kubeconfigResult)
	clientgoutils.Must(err)
	return
}

func KillSyncthingProcess(cli runner.Client) error {
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
	return util.Retry("kill syncthing process", funcs)
}

func Get(cli runner.Client) error {
	cases := []struct {
		resource string
		appName  string
		keywords []string
	}{
		{resource: "deployments", appName: "bookinfo", keywords: []string{"details", "productpage", "ratings", "reviews"}},
		{resource: "jobs", appName: "bookinfo", keywords: []string{"print-num-01"}},
		{resource: "service", appName: "bookinfo", keywords: []string{"details", "productpage", "ratings", "reviews"}},
		{resource: "pods", appName: "", keywords: []string{"details", "productpage", "ratings", "reviews"}},
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
	return util.Retry("get", funcs)
}

func TestLog(_ runner.Client) error {
	file := fp.NewFilePath(homedir.HomeDir()).
		RelOrAbs(".nh").
		RelOrAbs("nhctl").
		RelOrAbs("logs").
		RelOrAbs("nhctl.log").
		ReadFile()
	if len(file) == 0 {
		return errors.New("Daemon log file is empty, please check your log initialize code !!!")
	}
	return nil
}
