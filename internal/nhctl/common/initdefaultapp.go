/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package common

import (
	"fmt"
	errors2 "github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func InitDefaultApplicationInCurrentNs(appName, namespace, kubeconfigPath string) (*app.Application, error) {
	var err error
	fakedMeta := appmeta.FakeAppMeta(namespace, appName)
	if err := fakedMeta.InitGoClient(kubeconfigPath); err != nil {
		return nil, err
	}

	// continue initial when secret already exist
	if err := fakedMeta.Initial(true); err != nil && !k8serrors.IsAlreadyExists(err) {
		log.Logf("Init default.application meta failed: %v", err)
		return app.NewApplicationM(appName, namespace, kubeconfigPath, fakedMeta, true)
	}

	var actualMeta *appmeta.ApplicationMeta
	// re get from daemon
	if actualMeta, err = nocalhost.GetApplicationMeta(appName, namespace, kubeconfigPath); err != nil {
		log.Logf("Get default.application meta failed: %v", err)
		return app.NewApplicationM(appName, namespace, kubeconfigPath, fakedMeta, true)
	}

	if actualMeta.IsInstalled() {
		return app.NewApplicationM(appName, namespace, kubeconfigPath, actualMeta, true)
	}

	// set status as INSTALLED if not installed
	actualMeta.ApplicationState = appmeta.INSTALLED
	if err := actualMeta.Update(); err != nil {
		return app.NewApplicationM(appName, namespace, kubeconfigPath, actualMeta, true)
	}
	return app.NewApplicationM(appName, namespace, kubeconfigPath, actualMeta, true)
}

func InstallApplication(flags *app_flags.InstallFlags, applicationName, kubeconfig, namespace string) (*app.Application, error) {
	var err error

	// build Application will create the application meta and it's secret
	// init the application's config
	nocalhostApp, err := app.BuildApplication(applicationName, flags, kubeconfig, namespace)
	if err != nil {
		return nil, err
	}

	// if init appMeta successful, then should remove all things while fail
	defer func() {
		if err != nil {
			coloredoutput.Fail("Install application fail, try to rollback..")
			log.ErrorE(err, "")
			if err := nocalhostApp.Uninstall(true); err != nil {
				coloredoutput.Fail("Try uninstall fail, nocalhost will uninstall in force (There may be some residue in k8s)")
				utils.Should(nocalhostApp.Uninstall(true))
				coloredoutput.Success("Rollback success (There may be some residue in k8s)")
			} else {
				coloredoutput.Success("Rollback success")
			}
		}
	}()

	appType := nocalhostApp.GetType()
	if appType == "" {
		return nil, errors2.New("--type must be specified")
	}

	// add helmValue in config
	helmValue := nocalhostApp.GetApplicationConfigV2().HelmValues
	for _, v := range helmValue {
		flags.HelmSet = append([]string{fmt.Sprintf("%s=%s", v.Key, v.Value)}, flags.HelmSet...)
	}

	// the values.yaml config in nocalhost is the most highest priority
	// -f in helm, append it to the last
	vals := nocalhostApp.GetApplicationConfigV2().HelmVals
	if vals != nil && vals != "" {
		helmvals := fp.NewRandomTempPath().MkdirThen().RelOrAbs("nocalhost.helmvals")

		if marshal, err := yaml.Marshal(vals); err != nil {
			return nil, err
		} else {
			if err := helmvals.WriteFile(string(marshal)); err != nil {
				return nil, err
			}
			flags.HelmValueFile = append(flags.HelmValueFile, helmvals.Abs())
		}
	}

	flag := &app.HelmFlags{
		Values:   flags.HelmValueFile,
		Set:      flags.HelmSet,
		Wait:     flags.HelmWait,
		Chart:    flags.HelmChartName,
		RepoUrl:  flags.HelmRepoUrl,
		RepoName: flags.HelmRepoName,
		Version:  flags.HelmRepoVersion,
	}

	err = nocalhostApp.Install(flag)
	return nocalhostApp, err
}
