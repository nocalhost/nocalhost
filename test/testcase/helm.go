package testcase

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallBookInfoUseHelmVals install bookinfo use .nocalhost cfg:
//
// application:
//  helmVals:
//    service:
//      port: 9082
//
//    bookinfo:
//      deploy:
//        resources:
//          limits:
//            cpu: 1m
//            memory: 1Mi
//          requests:
//            cpu: 1m
//            memory: 1Mi
//
// and should make sure helm's template is correctly rendered
func InstallBookInfoUseHelmVals(c runner.Client, branch string) error {
	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
		c.GetNhctl().Command(
			context.Background(), "install", "bookinfohelm",
			"-u", "https://github.com/nocalhost/bookinfo.git", "-t",
			"helmGit", "-r", branch, "--resource-path", "charts/bookinfo", "--config", "config.helm.helmvals.yaml",
		),
	)

	if err := runner.Runner.RunSimple(
		c.SuiteName(),
		c.GetKubectl().Command(context.Background(), "get", "deployment", "details", "-o", "yaml"),
		false,
		func(sout string) error {
			if !strings.Contains(sout, "- containerPort: 9082") {
				return errors.New(
					fmt.Sprintf(
						"deployment[details] should contains '- containerPort: 9082', but actually: %s", sout,
					),
				)
			}

			if !strings.Contains(sout, "memory: 1Mi") || !strings.Contains(sout, "cpu: 1m") {
				return errors.New(
					fmt.Sprintf(
						"deployment[details] should contains 'memory: 1Mi and cpu: 1Mi', but actually: %s", sout,
					),
				)
			}

			return nil
		},
	); err != nil {
		return err
	}

	return listBookInfoHelm(c, true)
}

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func InstallBookInfoWithNhctl(c runner.Client) error {
	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
		c.GetNhctl().Command(
			context.Background(), "install", "bookinfohelm",
			"-u", "https://github.com/nocalhost/bookinfo.git", "-t",
			"helmGit", "--resource-path", "charts/bookinfo",
		),
	)
	return listBookInfoHelm(c, true)
}

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func UninstallBookInfoWithNhctl(c runner.Client) error {
	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
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
		c.SuiteName(),
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
		c.SuiteName(),
		exec.Command(
			"git", "clone", "--depth",
			"1", "https://github.com/nocalhost/bookinfo.git",
			tmpDir,
		),
	)

	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
		c.GetHelm().Command(
			context.Background(), "dependency", "build", helmResourceDir,
		),
	)

	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
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
				c.SuiteName(),
				c.GetNhctl().Command(
					context.Background(), "list",
				),
			)

			helmResult, _, _ := runner.Runner.Run(
				c.SuiteName(),
				c.GetHelm().Command(
					context.Background(), "list",
				),
			)

			if exist &&
				!(strings.Contains(nhctlResult, "bookinfohelm") && strings.Contains(
					helmResult, "bookinfohelm",
				)) {
				return errors.New(
					fmt.Sprintf(
						"do not list application named bookinfohelm, \nhelmresult: \n%s nhctlresult \n%s", helmResult,
						nhctlResult,
					),
				)
			}

			if !exist &&
				(strings.Contains(nhctlResult, "bookinfohelm") || strings.Contains(
					helmResult, "bookinfohelm",
				)) {
				return errors.New(
					fmt.Sprintf(
						"bookinfohelm is not expect but listed, \nhelmresult: \n%s nhctlresult \n%s", helmResult,
						nhctlResult,
					),
				)
			}
			return nil
		},
	)
}
