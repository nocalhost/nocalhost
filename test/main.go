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
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/suite"
	"nocalhost/test/util"
	"os"
	"time"
)

func main() {
	//_ = os.Setenv("LocalTest", "true")

	start := time.Now()

	var v2 string
	var t suite.T

	if _, ok := os.LookupEnv("LocalTest"); ok {
		// For local test
		cli := nhctlcli.NewNhctl("nocalhost", utils.GetHomePath()+"/.kube/config")
		clientgoutils.Must(util.Init(cli))
		t = suite.T{Cli: cli}
		_ = os.Setenv("commit_id", "test")
	} else {
		cli, _, v2Tmp, cancelFunc := suite.Prepare()
		v2 = v2Tmp
		t = suite.T{Cli: cli, CleanFunc: cancelFunc}
	}

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
