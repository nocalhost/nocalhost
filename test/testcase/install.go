/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package testcase

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func InstallBookInfoDifferentType(nhctl runner.Client) error {
	_ = UninstallBookInfo(nhctl)
	f := []func() error{
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoHelmGit(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoKustomizeGit(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoHelmRepo(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		}, func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installHelmRepoWithCredential(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		}, func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installHelmRepoAlreadyExist(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoRawManifestLocal(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoKustomizeGitLocal(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoHelmLocal(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
		func() error {
			return util.TimeoutFunc(
				time.Minute*2, func() error {
					return installBookInfoRawManifest(nhctl)
				}, func() error {
					return UninstallBookInfo(nhctl)
				},
			)
		},
	}
	for i, bookinfoFunc := range f {
		err := util.RetryFunc(
			func() error {
				logger := log.TestLogger(nhctl.SuiteName())

				if err := bookinfoFunc(); err != nil {
					logger.Infof("Error on exec INSTALL function index: %v, Err: %s", strconv.Itoa(i), err)
					return errors.Wrap(err, "error on exec function index: "+strconv.Itoa(i))
				}
				if err := List(nhctl); err != nil {
					logger.Infof("Error on exec INSTALL function index: %v, Err: %s", strconv.Itoa(i), err)
					return err
				}
				if err := UninstallBookInfo(nhctl); err != nil {
					logger.Infof("Error on exec INSTALL function index: %v, Err: %s", strconv.Itoa(i), err)
					return err
				}
				return nil
			},
		)
		if err != nil {
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoRawManifestLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command(
		"git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(nhctl.SuiteName(), command); err != nil {
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoKustomizeGitLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command(
		"git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(nhctl.SuiteName(), command); err != nil {
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmLocal(nhctl runner.Client) error {
	dir, _ := os.MkdirTemp("", "")
	command := exec.Command(
		"git",
		"clone",
		"-b",
		"test-case",
		"https://github.com/nocalhost/bookinfo.git",
		dir,
	)
	if err := runner.Runner.RunWithCheckResult(nhctl.SuiteName(), command); err != nil {
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
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installBookInfoHelmRepo(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"-t",
		string(appmeta.HelmRepo),
		"--helm-chart-name",
		"helm/bookinfo",
		"--helm-repo-url",
		"https://nocalhost-helm.pkg.coding.net/nocalhost/helm",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installHelmRepoWithCredential(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"--helm-chart-name",
		"bookinfo",
		"-t",
		string(appmeta.HelmRepo),
		"--helm-repo-url",
		fmt.Sprintf("https://%s:%s@nocalhost-helm.pkg.coding.net/nocalhost/test",
			os.Getenv(util.HelmRepoUsername), os.Getenv(util.HelmRepoPassword)),
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func installHelmRepoAlreadyExist(nhctl runner.Client) error {
	_, _, _ = nhctl.GetHelm().RunWithRollingOut(context.TODO(), "repo",
		"add",
		"bookinfo",
		"https://nocalhost-helm.pkg.coding.net/nocalhost/test",
		"--username",
		os.Getenv(util.HelmRepoUsername),
		"--password",
		os.Getenv(util.HelmRepoPassword),
	)
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "install",
		"bookinfo",
		"--helm-chart-name",
		"bookinfo",
		"-t",
		string(appmeta.HelmRepo),
		"--helm-repo-url",
		"https://nocalhost-helm.pkg.coding.net/nocalhost/test",
	)
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}
