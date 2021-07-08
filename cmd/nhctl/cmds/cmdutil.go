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

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/nocalhost"
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
		if errors.Is(err, app.ErrNotFound) && appName == nocalhost.DefaultNocalhostApplication {
			// try init default application
			if _, err := common.InitDefaultApplicationInCurrentNs(nameSpace, kubeConfig); err != nil {
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
		log.FatalE(err, "please make sure kubeconfig path is reachable")
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
