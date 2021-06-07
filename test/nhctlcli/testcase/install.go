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
	"nocalhost/test/nhctlcli/runner"
)

func InstallBookInfo(nhctl runner.Client) error {
	return installBookInfoRawManifest(nhctl)
}

func InstallBookInfoThreeTimes(nhctl runner.Client) error {
	_ = UninstallBookInfo(nhctl)
	//installBookInfoHelmGit(nhctl)
	//List(nhctl)
	//UninstallBookInfo(nhctl)
	if err := installBookInfoKustomizeGit(nhctl); err != nil {
		return err
	}
	if err := List(nhctl); err != nil {
		return err
	}
	if err := UninstallBookInfo(nhctl); err != nil {
		return err
	}
	if err := installBookInfoRawManifest(nhctl); err != nil {
		return err
	}
	if err := List(nhctl); err != nil {
		return err
	}
	return nil
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

func installBookInfoHelmGit(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"-t",
		"helmGit",
		"--resource-path",
		"charts/bookinfo",
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
		"kustomizeGit",
		"--resource-path",
		"kustomize/base",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}
