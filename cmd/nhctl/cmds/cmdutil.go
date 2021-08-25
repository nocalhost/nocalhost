/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
)

func initApp(appName string) {
	must(initAppMutate(appName))
}

func initAppMutate(appName string) error {
	var err error
	must(Prepare())

	nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true)
	if err != nil {
		log.Logf("Get application %s on namespace %s occurs error: %v", appName, nameSpace, err)
		// if default application not found, try to create one
		if errors.Is(err, app.ErrNotFound) && appName == _const.DefaultNocalhostApplication {
			// try init default application
			_, err := common.InitDefaultApplicationInCurrentNs(nameSpace, kubeConfig)

			// if some one can not create default.application
			// we also return a fake default.application
			if k8serrors.IsForbidden(err) {

				// if current user has not permission to create secret, we also create a fake 'default.application'
				// app meta for him
				nocalhostApp, err = app.NewFakeApplication(appName, nameSpace, kubeConfig, true)
				return err
			}

			if err != nil {
				return errors.Wrap(err, "Error while create default application")
			}

			// then reNew nocalhostApp
			if nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true); err != nil {
				return errors.Wrap(err, "Error while init default application")
			}

		} else {
			return errors.New("Failed to get application info")
		}
	}
	log.AddField("APP", nocalhostApp.Name)
	return nil
}

func Prepare() error {
	if kubeConfig == "" { // use default config
		kubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	abs, err := filepath.Abs(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "please make sure kubeconfig path is reachable")
	}
	kubeConfig = abs

	if nameSpace == "" {
		if nameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(kubeConfig); err != nil {
			return err
		}
		if nameSpace == "" {
			return errors.New("--namespace or --kubeconfig mush be provided")
		}
	}

	log.Debugf("Nocalhost Prepare successful, getting kubeconfig from %s, namespace %s", kubeConfig, nameSpace)
	return nil
}

func initService(svcName string, svcType string) *controller.Controller {
	if svcName == "" {
		log.Fatal("please use -d to specify a k8s workload")
	}
	return nocalhostApp.Controller(svcName, base.SvcTypeOf(svcType))
}

func checkIfSvcExist(svcName string, svcType string) {
	nocalhostSvc = initService(svcName, svcType)
	_, err := nocalhostSvc.CheckIfExist()
	if err != nil {
		log.FatalE(err, fmt.Sprintf("Resource: %s-%s not found!", svcType, svcName))
	}
	log.AddField("SVC", svcName)
}

func initAppAndCheckIfSvcExist(appName string, svcName string, svcType string) {
	initApp(appName)
	checkIfSvcExist(svcName, svcType)
}
