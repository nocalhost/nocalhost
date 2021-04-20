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

func InstallBookInfo(nhctl *nhctlcli.CLI) {
	UninstallBookInfo(nhctl)
	installBookInfoRawManifest(nhctl)
}

func InstallBookInfoThreeTimes(nhctl *nhctlcli.CLI) {
	UninstallBookInfo(nhctl)
	//installBookInfoHelmGit(nhctl)
	//UninstallBookInfo(nhctl)
	installBookInfoKustomizeGit(nhctl)
	UninstallBookInfo(nhctl)
	installBookInfoRawManifest(nhctl)
}

func UninstallBookInfo(nhctl *nhctlcli.CLI) {
	nhctl.RunWithRollingOut(context.Background(), "uninstall", "bookinfo", "--force")
}

func installBookInfoRawManifest(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		"rawManifest",
		"--resource-path",
		"manifest/templates")
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmGit(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		"helmGit",
		"--resource-path",
		"charts/bookinfo")
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoKustomizeGit(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		"kustomizeGit",
		"--resource-path",
		"kustomize/base")
	stdout, stderr, err := nhctlcli.Runner.RunWithRollingOut(cmd)
	nhctlcli.Runner.CheckResult(cmd, stdout, stderr, err)
}
