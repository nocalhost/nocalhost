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
	"fmt"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

func InitApp(appName string) {
	var err error

	if !nocalhost.CheckIfApplicationExist(appName) {
		log.FatalE(err, fmt.Sprintf("Application \"%s\" not found", appName))
	}
	nocalhostApp, err = app.NewApplication(appName)
	if err != nil {
		log.FatalE(err, "Failed to get application info")
	}
	log.AddField("APP", nocalhostApp.Name)
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

func InitAppAndCheckIfSvcExist(appName string, svcName string, svcAttr ...string) {
	serviceType := "deployment"
	if len(svcAttr) > 0 {
		serviceType = svcAttr[0]
	}
	InitApp(appName)
	CheckIfSvcExist(svcName, serviceType)
}
