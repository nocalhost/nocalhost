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
	"sigs.k8s.io/yaml"
	"time"
)

func Reset(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "reset", "bookinfo")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func Upgrade(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "upgrade",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"--resource-path",
		"manifest/templates")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func Profile(nhctl *nhctlcli.CLI) error {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.Command(ctx, "profile",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"--resource-path",
		"manifest/templates")
	nhctlcli.Runner.Run(cmd)
	return
}

func Config(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "config", "get", "bookinfo")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func SyncStatus(nhctl *nhctlcli.CLI, module string) error {
	cmd := nhctl.Command(context.Background(), "sync-status", "bookinfo", "-d", module)
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func List(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "list", "bookinfo")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func Db(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "db", "size", "--app", "bookinfo")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func Pvc(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "pvc", "list")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func NhctlVersion(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "version")
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	return nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}

func Apply(nhctl *nhctlcli.CLI) error {
	content := `{
	"apiVersion": "v1",
	"kind": "Service",
	"metadata": {
		"name": "reviews",
	"labels": {
		"app": "reviews",
		"service": "reviews",
		"apply": "apply"
		}
	},
	"spec": {
	"ports": [
		{
		"port": 9080,
		"name": "http"
		}
	],
	"selector": {
		"app": "reviews"
		}
	}
}`
	b, _ := yaml.JSONToYAML([]byte(content))
	f, _ := ioutil.TempFile("", "*apply.yaml")
	_, _ = f.Write(b)
	_ = f.Sync()
	defer f.Close()

	cmd := nhctl.Command(context.Background(), "apply", "bookinfo", f.Name())
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	return nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}
