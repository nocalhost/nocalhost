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

package suite

import (
	"nocalhost/test/nhctlcli"
	"nocalhost/test/nhctlcli/testcase"
)

func PortForward(cli *nhctlcli.CLI, _ ...string) {
	module := "reviews"
	port := 49080
	testcase.PortForwardStart(cli, module, port)
	testcase.PortForwardCheck(port)
	testcase.StatusCheckPortForward(cli, module, port)
	testcase.PortForwardEnd(cli, module, port)
}

func Dev(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	testcase.DevStart(cli, module)
	testcase.DevEnd(cli, module)
}

func Sync(cli *nhctlcli.CLI, _ ...string) {
	module := "ratings"
	testcase.DevStart(cli, module)
	testcase.Sync(cli, module)
	testcase.SyncStatus(cli, module)
	testcase.SyncCheck(cli, module)
	testcase.DevEnd(cli, module)
}

func Compatible(cli *nhctlcli.CLI, p ...string) {
	module := "ratings"
	port := 49080
	testcase.DevStart(cli, module)
	testcase.Sync(cli, module)
	testcase.PortForwardStart(cli, module, port)
	if len(p) > 0 && p[0] != "" {
		testcase.InstallNhctl(p[0])
		testcase.RestartDaemon(cli)
		testcase.NhctlVersion(cli)
	}
	testcase.StatusCheck(cli, module)
	testcase.SyncCheck(cli, module)
	testcase.PortForwardEnd(cli, module, port)
	testcase.DevEnd(cli, module)
	// for temporary
	//testcase.Upgrade(cli)
	testcase.Config(cli)
	testcase.List(cli)
	testcase.Db(cli)
	testcase.Pvc(cli)
	testcase.Reset(cli)
	testcase.InstallBookInfoThreeTimes(cli)
	testcase.Exec(cli)
}

func Reset(cli *nhctlcli.CLI, _ ...string) {
	testcase.Reset(cli)
	testcase.InstallBookInfo(cli)
}

func Upgrade(cli *nhctlcli.CLI, _ ...string) {
	testcase.InstallBookInfo(cli)
	testcase.Upgrade(cli)
}

func Install(cli *nhctlcli.CLI, _ ...string) {
	testcase.InstallBookInfoThreeTimes(cli)
	testcase.PortForwardCheck(39080)
}
