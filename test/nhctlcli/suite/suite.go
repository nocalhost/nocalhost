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
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/util"
	"os"
	"strings"
)

// test suite
type T struct {
	Cli       *nhctlcli.CLI
	CleanFunc func()
}

// Run command and clean environment after finished
func (t *T) Run(name string, fn func(cli *nhctlcli.CLI, p ...string), pp ...string) {
	defer func() {
		if err := recover(); err != nil {
			t.Clean()
			t.Alert()
			panic(err)
		}
	}()
	var retryTimes = 5
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfo(t.Cli); err != nil {
			_ = testcase.UninstallBookInfo(t.Cli)
			_ = testcase.Reset(t.Cli)
			continue
		}
		break
	}
	if err != nil {
		panic(errors.Wrap(err, "test suite failed, install bookinfo error"))
	}
	util.WaitResourceToBeStatus(t.Cli.Namespace, "pods", "app=reviews", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	util.WaitResourceToBeStatus(t.Cli.Namespace, "pods", "app=ratings", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	log.Info("Testing " + name)
	fn(t.Cli, pp...)
	log.Info("Testing done " + name)
	//testcase.Reset(t.Cli)
	for i := 0; i < retryTimes; i++ {
		if err = testcase.UninstallBookInfo(t.Cli); err != nil {
			continue
		}
		break
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
