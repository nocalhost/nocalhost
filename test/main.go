/*
Copyright 2021 The Nocalhost Authors.
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

package main

import (
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/suite"
	"nocalhost/test/nhctlcli/testcase"
	"nocalhost/test/util"
	"os"
	"time"
)

func main() {
	go util.TimeoutChecker(1 * time.Hour)
	v1, v2 := testcase.GetVersion()
	testcase.InstallNhctl(v1)
	path := os.Getenv("KUBECONFIG_PATH")
	if path == "" {
		path = "/root/.kube/config"
	}
	cli := nhctlcli.NewNhctl(path, "test")
	util.Init(cli)
	testcase.NhctlVersion(cli)
	testcase.StopDaemon(cli)
	go testcase.Init(cli)
	if i := <-testcase.StatusChan; i != 0 {
		testcase.StopChan <- 1
	}
	// ---------base line-----------
	t := suite.T{Cli: cli}
	t.Run("install", suite.Install)
	t.Run("dev", suite.Dev)
	t.Run("port-forward", suite.PortForward)
	t.Run("sync", suite.Sync)
	t.Run("upgrade", suite.Upgrade)
	t.Run("reset", suite.Reset)
	t.Run("compatible", suite.Compatible, v2)
	t.Clean()
}
