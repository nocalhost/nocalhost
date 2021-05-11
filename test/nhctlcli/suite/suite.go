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
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/util"
	"time"
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
			panic(err)
		}
	}()
	var retryTimes = 5
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = testcase.InstallBookInfo(t.Cli); err != nil {
			_ = testcase.Reset(t.Cli)
			time.Sleep(time.Second * 2)
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
			time.Sleep(time.Second * 2)
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
