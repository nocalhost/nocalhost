package testcase

import (
	"context"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func InstallBookInfoWithNhctl(c runner.Client) error {
	_ = runner.Runner.RunWithCheckResult(
		c.GetNhctl().Command(
			context.Background(), "install", "bookinfohelm",
			"-u", "https://github.com/anurnomeru/bookinfo.git", "-t",
			"helmGit", "--resource-path", "charts/bookinfo",
		),
	)
	return listBookInfoHelm(c, true)
}

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func UninstallBookInfoWithNhctl(c runner.Client) error {
	_ = runner.Runner.RunWithCheckResult(
		c.GetNhctl().Command(
			context.Background(), "uninstall", "bookinfohelm",
		),
	)
	return listBookInfoHelm(c, false)
}

// use helm uninstall to uninstall bookinfohelm,
// then check the result on nhctl and helm
func UninstallBookInfoWithNativeHelm(c runner.Client) error {
	_ = runner.Runner.RunWithCheckResult(
		c.GetHelm().Command(
			context.Background(), "uninstall", "bookinfohelm",
		),
	)
	return listBookInfoHelm(c, false)
}

// use helm install to install bookinfohelm,
// then check the result on nhctl and helm
func InstallBookInfoWithNativeHelm(c runner.Client) error {
	tmpDir, _ := ioutil.TempDir("", "")
	_ = os.MkdirAll(tmpDir, 0644)

	helmResourceDir := filepath.Join(tmpDir, "charts/bookinfo")

	_ = runner.Runner.RunWithCheckResult(
		exec.Command(
			"git", "clone", "--depth",
			"1", "https://github.com/anurnomeru/bookinfo.git",
			tmpDir,
		),
	)

	_ = runner.Runner.RunWithCheckResult(
		c.GetHelm().Command(
			context.Background(), "dependency", "build", helmResourceDir,
		),
	)

	_ = runner.Runner.RunWithCheckResult(
		c.GetHelm().Command(
			context.Background(), "install", "bookinfohelm", helmResourceDir,
		),
	)

	return listBookInfoHelm(c, true)
}

// while helm application is installed,
// then the result should be apply to both nhctl list or helm list
//
// also when a helm application is uninstalled,
// then either helm nor nhctl can not list it out.
func listBookInfoHelm(c runner.Client, exist bool) error {
	return util.RetryFunc(
		func() error {
			nhctlResult, _, _ := runner.Runner.Run(
				c.GetNhctl().Command(
					context.Background(), "list",
				),
			)

			helmResult, _, _ := runner.Runner.Run(
				c.GetHelm().Command(
					context.Background(), "list",
				),
			)

			if exist &&
				!(strings.Contains(nhctlResult, "bookinfohelm") && strings.Contains(
					helmResult, "bookinfohelm",
				)) {
				return errors.New("do not list application named bookinfohelm")
			}

			if !exist &&
				(strings.Contains(nhctlResult, "bookinfohelm") || strings.Contains(
					helmResult, "bookinfohelm",
				)) {
				return errors.New("bookinfohelm is not expect but listed")
			}
			return nil
		},
	)
}
