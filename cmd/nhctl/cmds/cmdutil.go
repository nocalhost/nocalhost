/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"errors"
	"fmt"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
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

	if err := Prepare(); err != nil {
		log.FatalE(err, "")
	}

	nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true)

	// if default application not found, try to creat one
	if err != nil && errors.Is(err, app.ErrNotFound) && appName == app.DefaultNocalhostApplication {

		// try init default application
		if err := InitDefaultApplicationInCurrentNs(); err != nil {
			log.FatalE(err, "Error while create default application")
		}

		// then reNew nocalhostApp
		if nocalhostApp, err = app.NewApplication(appName, nameSpace, kubeConfig, true); err != nil {
			log.FatalE(err, "Error while init default application")
		}

	} else {
		log.FatalE(err, "Failed to get application info")
	}
	log.AddField("APP", nocalhostApp.Name)
}

func Prepare() error {
	if kubeConfig == "" { // use default config
		kubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	var err error
	if nameSpace == "" {
		nameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(kubeConfig)
		if err != nil {
			return err
		}
		if nameSpace == "" {
			return errors.New("--namespace or --kubeconfig mush be provided")
		}
	}

	return nil
}

func CheckIfSvcExist(svcName string, svcType ...string) {
	serviceType := app.Deployment
	if len(svcType) > 0 {
		svcTypeLower := strings.ToLower(svcType[0])
		switch svcTypeLower {
		case strings.ToLower(string(app.StatefulSet)):
			serviceType = app.StatefulSet
		case strings.ToLower(string(app.DaemonSet)):
			serviceType = app.DaemonSet
		case strings.ToLower(string(app.Job)):
			serviceType = app.Job
		case strings.ToLower(string(app.CronJob)):
			serviceType = app.CronJob
		default:
			serviceType = app.Deployment
		}
	}
	if svcName == "" {
		log.Fatal("please use -d to specify a k8s workload")
	}
	exist, err := nocalhostApp.CheckIfSvcExist(svcName, serviceType)
	if err != nil {
		log.FatalE(err, fmt.Sprintf("failed to check if svc exists: %s", err.Error()))
	} else if !exist {
		log.Fatalf("\"%s\" not found", svcName)
	}

	log.AddField("SVC", svcName)
}

func initAppAndCheckIfSvcExist(appName string, svcName string, svcAttr []string) {
	serviceType := "deployment"
	if len(svcAttr) > 0 {
		serviceType = svcAttr[0]
	}
	initApp(appName)
	CheckIfSvcExist(svcName, serviceType)
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

	if err := InstallApplication(app.DefaultNocalhostApplication); k8serrors.IsServerTimeout(err) {
		return nil
	}
	return err
}
