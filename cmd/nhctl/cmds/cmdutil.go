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
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
)

func initApp(appName string) {
	var err error

	must(Prepare())

	nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true)
	if err != nil {
		// if default application not found, try to creat one
		if errors.Is(err, app.ErrNotFound) && appName == app.DefaultNocalhostApplication {
			// try init default application
			mustI(InitDefaultApplicationInCurrentNs(), "Error while create default application")

			// then reNew nocalhostApp
			nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true)
			mustI(err, "Error while init default application")

		} else {
			log.FatalE(err, "Failed to get application info")
		}
	}
	log.AddField("APP", nocalhostApp.Name)
}

func Prepare() error {
	if kubeConfig == "" { // use default config
		kubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	var err error
	if nameSpace == "" {
		if nameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(kubeConfig); err != nil {
			return err
		}
		if nameSpace == "" {
			return errors.New("--namespace or --kubeconfig mush be provided")
		}
	}

	return nil
}

func initService(svcName string, svcType string) *controller.Controller {
	if svcName == "" {
		log.Fatal("please use -d to specify a k8s workload")
	}

	serviceType := appmeta.Deployment
	if svcType != "" {
		svcTypeLower := strings.ToLower(svcType)
		switch svcTypeLower {
		case strings.ToLower(string(appmeta.Deployment)):
			serviceType = appmeta.Deployment
		case strings.ToLower(string(appmeta.StatefulSet)):
			serviceType = appmeta.StatefulSet
		case strings.ToLower(string(appmeta.DaemonSet)):
			serviceType = appmeta.DaemonSet
		case strings.ToLower(string(appmeta.Job)):
			serviceType = appmeta.Job
		case strings.ToLower(string(appmeta.CronJob)):
			serviceType = appmeta.CronJob
		default:
			log.FatalE(errors.New(fmt.Sprintf("Unsupported SvcType %s", svcType)), "")
		}
	}
	return nocalhostApp.Controller(svcName, serviceType)
}

func checkIfSvcExist(svcName string, svcType string) {
	nocalhostSvc = initService(svcName, svcType)
	exist, err := nocalhostSvc.CheckIfExist()
	if err != nil {
		log.FatalE(err, "failed to check if controller exists")
	} else if !exist {
		log.FatalE(errors.New(fmt.Sprintf("Service %s-%s not found!", string(serviceType), svcName)), "")
	}
	log.AddField("SVC", svcName)
}

func initAppAndCheckIfSvcExist(appName string, svcName string, svcType string) {
	initApp(appName)
	checkIfSvcExist(svcName, svcType)
}

func InitDefaultApplicationInCurrentNs() error {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	baseDir := fp.NewFilePath(tmpDir)
	nocalhostDir := baseDir.RelOrAbs(app.DefaultGitNocalhostDir)
	err = nocalhostDir.Mkdir()
	if err != nil {
		return err
	}

	var cfg = ".default_config"

	err = nocalhostDir.RelOrAbs(cfg).WriteFile("name: nocalhost.default\nmanifestType: rawManifestLocal")
	if err != nil {
		return err
	}

	installFlags.Config = cfg
	installFlags.AppType = string(appmeta.Manifest)
	installFlags.LocalPath = baseDir.Abs()

	if err = InstallApplication(app.DefaultNocalhostApplication); k8serrors.IsServerTimeout(err) {
		return nil
	}
	return err
}
