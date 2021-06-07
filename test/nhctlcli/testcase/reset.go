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
	"nocalhost/test/nhctlcli/runner"
	"sigs.k8s.io/yaml"
)

func Reset(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "reset", "bookinfo")
	return runner.Runner.RunWithCheckResult(cmd)
}

func Upgrade(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "upgrade",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"--resource-path",
		"manifest/templates")
	return runner.Runner.RunWithCheckResult(cmd)
}

func Config(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "config", "get", "bookinfo")
	return runner.Runner.RunWithCheckResult(cmd)
}

func SyncStatus(nhctl runner.Client, module string) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "sync-status", "bookinfo", "-d", module)
	return runner.Runner.RunWithCheckResult(cmd)
}

func List(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "list", "bookinfo")
	return runner.Runner.RunWithCheckResult(cmd)
}

func Db(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "db", "size", "--app", "bookinfo")
	return runner.Runner.RunWithCheckResult(cmd)
}

func Pvc(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "pvc", "list")
	return runner.Runner.RunWithCheckResult(cmd)
}

func NhctlVersion(nhctl *runner.CLI) error {
	cmd := nhctl.Command(context.Background(), "version")
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func Apply(nhctl runner.Client) error {
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

	cmd := nhctl.GetNhctl().Command(context.Background(), "apply", "bookinfo", f.Name())
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}
