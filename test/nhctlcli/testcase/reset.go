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

package testcase

import (
	"context"
	"nocalhost/test/nhctlcli"
)

func Reset(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "reset", "bookinfo")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func Upgrade(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "upgrade",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"--resource-path",
		"manifest/templates")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func Config(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "config", "get", "bookinfo")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func SyncStatus(nhctl *nhctlcli.CLI, module string) {
	cmd := nhctl.Command(context.Background(), "sync-status", "bookinfo", "-d", module)
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func List(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "list", "bookinfo")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func Db(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "db", "size", "--app", "bookinfo")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func Pvc(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "pvc", "list")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func NhctlVersion(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "version")
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}
