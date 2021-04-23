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
	"io/ioutil"
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

func Apply(nhctl *nhctlcli.CLI) {
	content := `---
apiVersion: v1
kind: Service
metadata:
  name: reviews
  labels:
    app: reviews
    service: reviews
    apply: apply
spec:
  ports:
  - port: 9080
    name: http
  selector:
    app: reviews
`
	f, _ := ioutil.TempFile("/tmp", "apply.yaml")
	_, _ = f.WriteString(content)
	_ = f.Sync()
	defer f.Close()

	cmd := nhctl.Command(context.Background(), "apply", "bookinfo", f.Name())
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}
