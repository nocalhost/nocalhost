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

package testcase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/util"
	"os"
	"strings"
	"time"
)

func DevStart(cli *nhctlcli.CLI, moduleName string) {
	if err := os.MkdirAll(fmt.Sprintf("/tmp/%s", moduleName), 0777); err != nil {
		panic(fmt.Sprintf("test case failed, reason: create directory error, error: %v\n", err))
	}
	cmd := cli.Command(context.Background(), "dev",
		"start",
		"bookinfo",
		"-d", moduleName,
		"-s", "/tmp/"+moduleName,
		"--priority-class", "nocalhost-container-critical")
	nhctlcli.Runner.RunPanicIfError(cmd)

	util.WaitToBeStatus("test", "pods", "app="+moduleName, func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
}

func Sync(cli *nhctlcli.CLI, moduleName string) {
	cmd := cli.Command(context.Background(), "sync", "bookinfo", "-d", moduleName)
	nhctlcli.Runner.RunPanicIfError(cmd)
}
func SyncCheck(cli *nhctlcli.CLI, moduleName string) {
	filename := "hello.test"
	content := "this is a test, random string: " + uuid.New().String()
	if err := ioutil.WriteFile(
		fmt.Sprintf("/tmp/%s/%s", moduleName, filename),
		[]byte(content), 0644,
	); err != nil {
		panic(fmt.Sprintf("test case failed, reason: write file %s error: %v", filename, err))
	}
	// wait file to be synchronize
	time.Sleep(30 * time.Second)
	// not use nhctl exec is just because nhctl exec will stuck while cat file
	cmd := fmt.Sprintf(
		"kubectl exec -t deployment/%s -n %s cat %s --kubeconfig=%s",
		moduleName, cli.Namespace, filename, cli.KubeConfig,
	)
	fmt.Println("Running command: " + cmd)
	ok, log := util.WaitForCommandDone(cmd)
	if !ok {
		panic(
			fmt.Sprintf(
				"test case failed, reason: cat file %s error,"+
					" command: %s, log: %v", filename, cmd, log,
			),
		)
	}
	if !strings.Contains(log, content) {
		panic(
			fmt.Sprintf(
				"test case failed, reason: file content:"+
					" %s not equals command log: %s", content, log,
			),
		)
	}
}

func PortForwardCheck(port int) {
	retry := 100
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/health", port), nil)
	for i := 0; i < retry; i++ {
		res, _ := http.DefaultClient.Do(req)
		if res == nil || res.StatusCode != 200 {
			time.Sleep(2 * time.Second)
		} else {
			return
		}
	}
	panic("test case failed, reason: can't access endpoint: " + fmt.Sprintf("http://localhost:%d/health", port))
}

func DevEnd(cli *nhctlcli.CLI, moduleName string) {
	cmd := cli.Command(context.Background(), "dev", "end", "bookinfo", "-d", moduleName)
	nhctlcli.Runner.RunPanicIfError(cmd)

	util.WaitToBeStatus("test", "pods", "app="+moduleName, func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning && func() bool {
			for _, containerStatus := range i.(*v1.Pod).Status.ContainerStatuses {
				if containerStatus.Ready {
					return false
				}
			}
			return true
		}()
	})
}
