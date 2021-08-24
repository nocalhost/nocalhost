package testcase

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"strings"
	"sync"
	"time"
)

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func InstallBookInfoHelmForTestHook(c runner.Client) error {
	_, _, _ = runner.Runner.RunWithRollingOutWithChecker(
		c.SuiteName(),
		c.GetNhctl().Command(
			context.Background(), "install", "bookinfohelm",
			"-u", "https://github.com/nocalhost/bookinfo.git", "-t",
			"rawManifest", "--config", "config.yaml", "-r", "test-hook",
		), nil,
	)
	return ShouldHaveJob(c, "pre-install", "post-install")
}

func UpgradeBookInfoHelmForTestHook(c runner.Client) error {
	_, _, _ = runner.Runner.RunWithRollingOutWithChecker(
		c.SuiteName(),
		c.GetNhctl().Command(
			context.Background(), "upgrade", "bookinfohelm",
			"--git-url", "https://github.com/nocalhost/bookinfo.git",
			"--config", "config.yaml", "-r", "test-hook",
		), nil,
	)
	return ShouldHaveJob(c, "pre-upgrade", "post-upgrade")
}

// use nhctl install to install bookinfohelm,
// then check the result on nhctl and helm
func UninstallBookInfoHelmForTestHook(c runner.Client) error {
	wg := sync.WaitGroup{}
	wg.Add(2)

	logger := log.TestLogger(c.SuiteName())

	errorChan := make(chan error, 0)
	successChan := make(chan interface{}, 0)
	go func() {
		logger.Info("Wait for job pre-uninstall")
		if err := k8sutils.WaitPod(
			c.GetClientset(),
			c.GetNhctl().Namespace,
			metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("job-name", "pre-delete").String()},
			func(i *v1.Pod) bool { return i.Status.Phase == v1.PodSucceeded },
			time.Hour*1,
		); err != nil {
			errorChan <- err
		}
		wg.Done()
		logger.Info("Wait for job pre-uninstall complete")
	}()

	go func() {
		logger.Info("Wait for job post-uninstall")
		if err := k8sutils.WaitPod(
			c.GetClientset(),
			c.GetNhctl().Namespace,
			metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("job-name", "post-delete").String()},
			func(i *v1.Pod) bool { return i.Status.Phase == v1.PodSucceeded },
			time.Hour*1,
		); err != nil {
			errorChan <- err
		}
		wg.Done()
		logger.Info("Wait for job post-uninstall complete")
	}()

	go func() {
		wg.Wait()
		successChan <- "Success"
	}()

	_ = runner.Runner.RunWithCheckResult(
		c.SuiteName(),
		c.GetNhctl().Command(
			context.Background(), "uninstall", "bookinfohelm",
		),
	)

	select {
	case <-time.Tick(5 * time.Minute):
		return errors.New("Timeout while uninstall bookinfo")
	case e := <-errorChan:
		return e
	case <-successChan:
		return ShouldNotHaveAnyJob(c)
	}
}

func ShouldNotHaveAnyJob(c runner.Client) error {
	return runner.Runner.RunSimple(
		c.SuiteName(),
		c.GetKubectl().Command(context.Background(), "get", "job"),
		true,
		func(sout string) error {
			if strings.TrimSpace(sout) != "" {
				return errors.New(
					fmt.Sprintf(
						"should not contain any job, sout: %s", sout,
					),
				)
			}
			return nil
		},
	)
}

func ShouldHaveJob(c runner.Client, jobNames ...string) error {
	return runner.Runner.RunSimple(
		c.SuiteName(),
		c.GetKubectl().Command(context.Background(), "get", "job"),
		true,
		func(sout string) error {
			for _, name := range jobNames {
				if !strings.Contains(sout, name) {
					return errors.New(
						fmt.Sprintf(
							"job '%s' should be run, sout: %s", name, sout,
						),
					)
				}
			}

			return nil
		},
	)
}
