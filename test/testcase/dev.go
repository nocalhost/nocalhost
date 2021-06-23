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
	"github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"net/http"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/test/runner"
	"os"
	"strings"
	"time"
)

func DevStart(cli runner.Client, moduleName string) error {
	return DevStartT(cli, moduleName, "")
}

func DevStartT(cli runner.Client, moduleName string, moduleType string) error {
	syncDir := fmt.Sprintf("/tmp/%s/%s", cli.NameSpace(), moduleName)

	if err := os.MkdirAll(syncDir, 0777); err != nil {
		return errors.Errorf("test case failed, reason: create directory error, error: %v", err)
	}
	cmd := cli.GetNhctl().Command(
		context.Background(), "dev",
		"start",
		"bookinfo",
		"-d", moduleName,
		"-s", syncDir,
		"-t", moduleType,
		"--priority-class", "nocalhost-container-critical",
		// prevent tty to block testcase
		"--without-terminal",
	)
	if err := runner.Runner.RunWithCheckResult(cmd); err != nil {
		return err
	}
	_ = k8sutils.WaitPod(
		cli.GetClientset(),
		cli.GetNhctl().Namespace,
		metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", moduleName).String()},
		func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
		time.Minute*2,
	)
	return nil
}

func Sync(cli runner.Client, moduleName string) error {
	return SyncT(cli, moduleName, "")
}

func SyncT(cli runner.Client, moduleName string, moduleType string) error {
	cmd := cli.GetNhctl().Command(context.Background(), "sync", "bookinfo", "-d", moduleName, "-t", moduleType)
	return runner.Runner.RunWithCheckResult(cmd)
}

func SyncCheck(cli runner.Client, moduleName string) error {
	return SyncCheckT(cli, cli.NameSpace(), moduleName, "deployment")
}

func SyncCheckT(cli runner.Client, ns, moduleName string, moduleType string) error {
	filename := "hello.test"
	syncFile := fmt.Sprintf("/tmp/%s/%s/%s", cli.NameSpace(), moduleName, filename)

	content := "this is a test, random string: " + uuid.New().String()
	if err := ioutil.WriteFile(syncFile, []byte(content), 0644); err != nil {
		return errors.Errorf("test case failed, reason: write file %s error: %v", filename, err)
	}
	// wait file to be synchronize
	time.Sleep(10 * time.Second)
	if moduleType == "" {
		moduleType = "deployment"
	}
	// not use nhctl exec is just because nhctl exec will stuck while cat file
	args := []string{
		"-t", fmt.Sprintf("%s/%s", moduleType, moduleName),
		"--",
		"cat",
		filename,
	}
	logStr, _, err := cli.GetKubectl().Run(context.Background(), "exec", args...)
	if err != nil {
		return errors.Errorf(
			"test case failed, reason: cat file %s error, command: %s, log: %v",
			filename, args, logStr,
		)
	}
	if !strings.Contains(logStr, content) {
		return errors.Errorf(
			"test case failed, reason: file content: %s not equals command log: %s",
			content, logStr,
		)
	}
	return nil
}

func PortForwardCheck(port int) error {
	retry := 100
	endpoint := fmt.Sprintf("http://localhost:%d/health", port)
	req, _ := http.NewRequest("GET", endpoint, nil)
	for i := 0; i < retry; i++ {
		res, _ := http.DefaultClient.Do(req)
		if res == nil || res.StatusCode != 200 {
			time.Sleep(2 * time.Second)
		} else {
			return nil
		}
	}
	return errors.Errorf("test case failed, reason: can't access endpoint: %s", endpoint)
}

func DevEnd(cli runner.Client, moduleName string) error {
	return DevEndT(cli, moduleName, "")
}

func DevEndT(cli runner.Client, moduleName string, moduleType string) error {
	cmd := cli.GetNhctl().Command(context.Background(), "dev", "end", "bookinfo", "-d", moduleName, "-t", moduleType)
	if err := runner.Runner.RunWithCheckResult(cmd); err != nil {
		return err
	}
	_ = k8sutils.WaitPod(
		cli.GetClientset(),
		cli.GetNhctl().Namespace,
		metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", moduleName).String()},
		func(i *v1.Pod) bool {
			return i.Status.Phase == v1.PodRunning && func() bool {
				for _, containerStatus := range i.Status.ContainerStatuses {
					if containerStatus.Ready {
						return false
					}
				}
				return true
			}()
		},
		time.Minute*2,
	)
	return nil
}
