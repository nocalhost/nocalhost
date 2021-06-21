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

package main

import (
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/suite"
	"nocalhost/test/testcase"
	"os"
	"path/filepath"
	"time"
)

func main() {
	//_ = os.Setenv("LocalTest", "true")

	start := time.Now()

	var v2 string
	var t *suite.T

	if _, ok := os.LookupEnv("LocalTest"); ok {
		// For local test
		t = suite.NewT("nocalhost", filepath.Join(utils.GetHomePath(), ".kube", "config"), nil)
		_ = os.Setenv("commit_id", "test")
	} else {
		cancelFunc, ns, kubeconfig := suite.Prepare()
		t = suite.NewT(ns, kubeconfig, cancelFunc)
		_, v2 = testcase.GetVersion()
	}

	t.Run("helm-adaption",suite.HelmAdaption)
	t.Run("install", suite.Install)
	t.Run("deployment", suite.Deployment)
	//t.Run("port-forward", suite.PortForward)
	//t.Run("port-forward service", suite.PortForwardService)
	//t.Run("sync", suite.Sync)
	t.Run("application", suite.Upgrade)
	//t.Run("reset", suite.Reset)
	//t.Run("apply", suite.Apply)
	//t.Run("profile", suite.Profile)
	t.Run("statefulSet", suite.StatefulSet)
	t.Run("compatible", suite.Compatible, v2)
	t.Clean()

	log.Infof("Total time: %v", time.Now().Sub(start).Seconds())
}
