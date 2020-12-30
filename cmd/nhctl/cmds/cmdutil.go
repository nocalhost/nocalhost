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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func InitApp(appName string) {
	var err error

	if !nocalhost.CheckIfApplicationExist(appName) {
		log.Fatalf("application \"%s\" not found", appName)
	}
	nocalhostApp, err = app.NewApplication(appName)
	if err != nil {
		log.Fatal("failed to get application info")
	}
}

func CheckIfSvcExist(svcName string) {
	if svcName == "" {
		log.Fatal("please use -d to specify a k8s workload")
	}

	exist, err := nocalhostApp.CheckIfSvcExist(svcName, app.Deployment)
	if err != nil {
		log.Fatalf("failed to check if svc exists: %s", err.Error())
	} else if !exist {
		log.Fatalf("\"%s\" not found", svcName)
	}
}

func InitAppAndCheckIfSvcExist(appName string, svcName string) {
	InitApp(appName)
	CheckIfSvcExist(svcName)
}
