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

package suite

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"net/http"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/testcase"
	"nocalhost/test/util"
	"os"
	"strings"
	"time"
)

// test suite
type T struct {
	Cli       runner.Client
	CleanFunc func()
}

func NewT(namespace, kubeconfig string, f func()) *T {
	return &T{
		Cli:       runner.NewClient(kubeconfig, namespace),
		CleanFunc: f,
	}
}

func (t *T) Run(name string, fn func(cli runner.Client, p ...string), pp ...string) {
	t.RunWithBookInfo(true, name, fn, pp...)
}

// Run command and clean environment after finished
func (t *T) RunWithBookInfo(withBookInfo bool, name string, fn func(cli runner.Client, p ...string), pp ...string) {
	log.Infof("\n============= Testing (Start)%s  =============\n", name)
	timeBefore := time.Now()

	defer func() {
		if err := recover(); err != nil {
			t.Clean()
			t.Alert()
			panic(err)
		}
	}()

	clientForRunner := t.Cli.RandomNsCli()
	if err := util.RetryFunc(
		func() error {
			result, errOutput, err := clientForRunner.GetKubectl().RunClusterScope(
				context.Background(), "create", "ns", clientForRunner.NameSpace(),
			)

			if k8serrors.IsAlreadyExists(err) || strings.Contains(errOutput, "already exists") {
				return nil
			}

			if strings.Contains(result, "created") {
				return nil
			}

			return errors.Wrap(err, "Error while create ns: "+errOutput)
		},
	); err != nil {
		panic(err)
		return
	}

	log.Infof("\n============= Testing (Create Ns)%s  =============\n", name)

	var retryTimes = 10
	if withBookInfo {
		var err error
		for i := 0; i < retryTimes; i++ {
			timeBeforeInstall := time.Now()
			log.Infof("\n============= Testing (Installing BookInfo %d)%s =============\n", i, name)
			timeoutCtx, _ := context.WithTimeout(context.Background(), 10*time.Minute)
			if err = testcase.InstallBookInfo(timeoutCtx, clientForRunner); err != nil {
				log.Infof("\n============= Testing (Install BookInfo Failed)%s =============, Err: \n", name, err.Error())
				_ = testcase.UninstallBookInfo(clientForRunner)
				_ = testcase.Reset(clientForRunner)
				continue
			}
			timeAfterInstall := time.Now()
			log.Infof("\n============= Testing (BookInfo Installed, Cost(%fs) %s =============\n",
				timeAfterInstall.Sub(timeBeforeInstall).Seconds(), name)
			break
		}

		if err != nil {
			panic(errors.Wrap(err, "test suite failed, install bookinfo error"))
		}

		_ = k8sutils.WaitPod(
			clientForRunner.GetClientset(),
			clientForRunner.GetNhctl().Namespace,
			metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "reviews").String()},
			func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
			time.Minute*5,
		)
		_ = k8sutils.WaitPod(
			clientForRunner.GetClientset(),
			clientForRunner.GetNhctl().Namespace,
			metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "ratings").String()},
			func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
			time.Minute*5,
		)
	}

	log.Infof("\n============= Testing (Test)%s =============\n", name)

	fn(clientForRunner, pp...)

	timeAfter := time.Now()
	log.Infof("\n============= Testing done, Cost(%fs) %s =============\n", timeAfter.Sub(timeBefore).Seconds(), name)

	if withBookInfo {
		//testcase.Reset(clientForRunner)
		for i := 0; i < retryTimes; i++ {
			if err := testcase.UninstallBookInfo(clientForRunner); err != nil {
				continue
			}
			break
		}
	}
}

func (t *T) Clean() {
	if t.CleanFunc != nil {
		t.CleanFunc()
	}
}

func (t *T) Alert() {
	if oldV, newV := testcase.GetVersion(); oldV != "" && newV != "" {
		if webhook := os.Getenv(util.TestcaseWebhook); webhook != "" {
			s := `{"msgtype":"text","text":{"content":"兼容性测试(%s --> %s)没通过，请相关同学注意啦!",
"mentioned_mobile_list":["18511859195"]}}`
			var req *http.Request
			var err error
			data := strings.NewReader(fmt.Sprintf(s, oldV, newV))
			if req, err = http.NewRequest("POST", webhook, data); err != nil {
				log.Info(err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if _, err = http.DefaultClient.Do(req); err != nil {
				log.Info(err)
			}
		}
	}
}
