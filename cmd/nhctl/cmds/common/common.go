/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package common

import (
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
)

var (
	ServiceType  = "deployment"
	WorkloadName string
	KubeConfig   string // the path to the kubeconfig file
	NameSpace    string

	NocalhostApp *app.Application
	//NocalhostSvc *controller.Controller
)

func InitAppAndCheckIfSvcExist(appName string, svcName string, svcType string) (*controller.Controller, error) {
	if err := InitApp(appName); err != nil {
		return nil, err
	}
	return InitAndCheckIfSvcExist(svcName, svcType)
}

func InitApp(appName string) error {
	return InitAppMutate(appName)
}

func InitAndCheckIfSvcExist(svcName string, svcType string) (*controller.Controller, error) {
	nocalhostSvc, err := InitService(svcName, svcType)
	if err != nil {
		return nil, err
	}
	return nocalhostSvc, nocalhostSvc.CheckIfExist()
	//log.AddField("SVC", svcName)
}

func InitAppMutate(appName string) error {
	var err error
	if err := Prepare(); err != nil {
		return err
	}

	NocalhostApp, err = app.NewApplication(appName, NameSpace, KubeConfig, true)
	if err != nil {
		log.Logf("Get application %s on namespace %s occurs error: %v", appName, NameSpace, err)
		// if default application not found, try to create one
		if errors.Is(err, app.ErrNotFound) && appName == _const.DefaultNocalhostApplication {
			NocalhostApp, err = common.InitDefaultApplicationInCurrentNs(appName, NameSpace, KubeConfig)
			return err
		} else {
			return errors.New("Failed to get application info")
		}
	}
	log.AddField("APP", NocalhostApp.Name)
	return nil
}

func InitService(svcName string, svcType string) (*controller.Controller, error) {
	if svcName == "" {
		return nil, errors.New("please use -d to specify a k8s workload")
	}
	st, err := nocalhost.SvcTypeOfMutate(svcType)
	if err != nil {
		return nil, err
	}
	return NocalhostApp.Controller(svcName, st)
}

func Prepare() error {
	if KubeConfig == "" { // use default config
		KubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	abs, err := filepath.Abs(KubeConfig)
	if err != nil {
		return errors.Wrap(err, "please make sure kubeconfig path is reachable")
	}
	KubeConfig = abs

	if NameSpace == "" {
		if NameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(KubeConfig); err != nil {
			return err
		}
		if NameSpace == "" {
			return errors.New("--namespace or --kubeconfig mush be provided")
		}
	}

	return nil
}

func Must(err error) {
	MustI(err, "")
}

func MustI(err error, info string) {
	if k8serrors.IsForbidden(err) {
		log.FatalE(err, "Permission Denied! Please check that"+
			" your ServiceAccount(KubeConfig) has appropriate permissions.\n\n")
	} else if err != nil {
		log.FatalE(err, info)
	}
}
