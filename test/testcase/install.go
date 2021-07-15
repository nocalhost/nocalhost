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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/test/runner"
	"os"
	"os/exec"
	"strconv"
)

func InstallBookInfoDifferentType(nhctl runner.Client) error {
	_ = UninstallBookInfo(nhctl)
	f := []func() error{
		func() error { return installBookInfoHelmGit(nhctl) },
		func() error { return installBookInfoKustomizeGit(nhctl) },
		func() error { return installBookInfoHelmRepo(nhctl) },
		func() error { return installBookInfoRawManifestLocal(nhctl) },
		func() error { return installBookInfoKustomizeGitLocal(nhctl) },
		func() error { return installBookInfoHelmLocal(nhctl) },
		func() error { return installBookInfoRawManifest(nhctl) },
	}
	for i, bookinfoFunc := range f {
		if err := bookinfoFunc(); err != nil {
			return errors.Wrap(err, "error on exec function index: "+strconv.Itoa(i))
		}
		if err := List(nhctl); err != nil {
			return err
		}
		if err := UninstallBookInfo(nhctl); err != nil {
			return err
		}
	}
	return nil
}

func InstallBookInfo(ctx context.Context, nhctl runner.Client) error {
	return installBookInfoRawManifest(nhctl)
}

func UninstallBookInfo(nhctl runner.Client) error {
	stdout, stderr, err := nhctl.GetNhctl().RunWithRollingOut(
		context.Background(), "uninstall", "bookinfo", "--force",
	)
	if err != nil {
		return errors.Errorf(
			"Run command uninstall bookinfo error: %v, stdout: %s, stderr: %s",
			err, stdout, stderr,
		)
	}
	return nil
}

func installBookInfoRawManifest(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t", "rawManifest",
		"-r", "test-case",
		"--resource-path",
		"manifest/templates",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoKustomizeGit(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		string(appmeta.KustomizeGit),
		"--resource-path",
		"kustomize/base",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmGit(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		string(appmeta.Helm),
		"--resource-path",
		"charts/bookinfo",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoRawManifestLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command("git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(command); err != nil {
		return err
	}

	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-t",
		string(appmeta.ManifestLocal),
		"--local-path",
		dir,
		"--resource-path",
		"manifest/templates",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoKustomizeGitLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command("git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(command); err != nil {
		return err
	}

	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-t",
		string(appmeta.KustomizeLocal),
		"--local-path",
		dir,
		"--resource-path",
		"kustomize/base",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command("git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(command); err != nil {
		return err
	}
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-t",
		string(appmeta.HelmLocal),
		"--local-path",
		dir,
		"--resource-path",
		"charts/bookinfo",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmRepo(nhctl runner.Client) error {
	// this is nocalhost bug, chart name not work
	temp, _ := os.CreateTemp("", "")
	_, _ = temp.WriteString("name: nocalhost-bookinfo")
	_ = temp.Sync()
	_ = temp.Close()
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-t",
		string(appmeta.HelmRepo),
		"--helm-chart-name",
		"bookinfo/nocalhost-bookinfo",
		"--helm-repo-url",
		"https://codingcorp-helm.pkg.coding.net/naison-test/bookinfo",
		"--outer-config",
		temp.Name(),
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}
