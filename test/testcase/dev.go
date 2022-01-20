/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package testcase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"strings"
	"time"
)

func DevStartDeployment(cli runner.Client, moduleName string) error {
	return DevStartT(cli, moduleName, "deployment", profile.ReplaceDevMode)
}

func DevStartStatefulSet(cli runner.Client, moduleName string) error {
	return DevStartT(cli, moduleName, "statefulset", profile.ReplaceDevMode)
}

func DevStartDeploymentDuplicate(cli runner.Client, moduleName string) error {
	return DevStartT(cli, moduleName, "deployment", profile.DuplicateDevMode)
}

func DevStartStatefulSetDuplicate(cli runner.Client, moduleName string) error {
	return DevStartT(cli, moduleName, "statefulset", profile.DuplicateDevMode)
}

func DevStartT(cli runner.Client, moduleName string, moduleType string, modeType profile.DevModeType) error {
	syncDir := fmt.Sprintf("/tmp/%s/%s", cli.NameSpace(), moduleName)

	if err := os.MkdirAll(syncDir, 0777); err != nil {
		return errors.Errorf("test case failed, reason: create directory error, error: %v", err)
	}
	_ = exec.Command("git", "clone", "--depth", "1", "https://github.com/nocalhost/bookinfo-details.git", syncDir).Run()

	cmd := cli.GetNhctl().Command(
		context.Background(), "dev",
		"start",
		"bookinfo",
		"-d", moduleName,
		"-s", syncDir,
		"-t", moduleType,
		"--dev-mode", modeType.ToString(),
		"--priority-class", "nocalhost-container-critical",
		// prevent tty to block testcase
		"--without-terminal",
		//"--sidecar-image",
		//"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/golang:latest",
	)
	if stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(
		cli.SuiteName(), cmd, nil,
	); runner.Runner.CheckResult(
		cmd, stdout, stderr, err,
	) != nil {
		return err
	}
	//_ = k8sutils.WaitPod(
	//	cli.GetClientset(),
	//	cli.GetNhctl().Namespace,
	//	metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", moduleName).String()},
	//	func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
	//	time.Minute*30,
	//)
	// wait for file to be synchronized
	cmd = cli.GetNhctl().Command(context.TODO(), "sync-status", "bookinfo", "-d", moduleName, "-t", moduleType, "--wait")
	_, _, _ = runner.Runner.RunWithRollingOutWithChecker("wait for author source code to be synchronized", cmd, nil)
	time.Sleep(time.Second * 5)
	//go func() {
	//	podName, _ := cli.GetNhctl().Command(
	//		context.Background(), "dev",
	//		"pod",
	//		"bookinfo",
	//		"-d", moduleName,
	//		"-t", moduleType,
	//	).Output()
	//	cmd = cli.GetKubectl().Command(context.TODO(), "exec", "pods/"+string(podName), "--", "sh -c /home/nocalhost-dev/run.sh")
	//	_, _, _ = runner.Runner.Run("startup main process", cmd)
	//}()
	return nil
}

func Sync(cli runner.Client, moduleName string) error {
	return SyncT(cli, moduleName, "deployment")
}

func SyncT(cli runner.Client, moduleName string, moduleType string) error {
	cmd := cli.GetNhctl().Command(context.Background(), "sync", "bookinfo", "-d", moduleName, "-t", moduleType)
	return runner.Runner.RunWithCheckResult(cli.SuiteName(), cmd)
}

func SyncCheck(cli runner.Client, moduleName string) error {
	return SyncCheckT(cli, moduleName, "deployment")
}

func SyncCheckT(cli runner.Client, moduleName, moduleType string) error {
	if moduleType == "" {
		moduleType = "deployment"
	}
	filename := "hello.test"
	syncFile := fmt.Sprintf("/tmp/%s/%s/%s", cli.NameSpace(), moduleName, filename)

	content := "this is a test, random string: " + uuid.New().String()
	if err := ioutil.WriteFile(syncFile, []byte(content), 0644); err != nil {
		return errors.Errorf("test case failed, reason: write file %s error: %v", filename, err)
	}

	return util.RetryFunc(
		func() error {
			// wait file to be synchronize
			time.Sleep(5 * time.Second)
			// not use nhctl exec is just because nhctl exec will stuck while cat file
			// get pod
			podName, _, _ := cli.GetNhctl().Run(context.TODO(), "dev", []string{
				"pod", "bookinfo", "-t", moduleType, "-d", moduleName}...)
			args := []string{
				"-t", fmt.Sprintf("pods/%s", podName),
				"-c",
				"nocalhost-dev",
				"--",
				"cat",
				filename,
			}
			logStr, errStr, err := cli.GetKubectl().Run(context.Background(), "exec", args...)
			if err != nil {
				return errors.Errorf(
					"test case failed, reason: cat file %s error, command: %s, stdout: %v, stderr: %v",
					filename, args, logStr, errStr,
				)
			}
			if !strings.Contains(logStr, content) {
				return errors.Errorf(
					"test case failed, reason: file content: %s not equals command log: %s",
					content, logStr,
				)
			}
			return nil
		},
	)
}

func PortForwardCheck(port int) error {
	retry := 100
	endpoint := fmt.Sprintf("http://localhost:%d/health", port)
	req, _ := http.NewRequest("GET", endpoint, nil)
	for i := 0; i < retry; i++ {
		res, _ := http.DefaultClient.Do(req)
		if res == nil || res.StatusCode != 200 {
			time.Sleep(2 * time.Second)
		} else {
			return nil
		}
	}
	return errors.Errorf("test case failed, reason: can't access endpoint: %s", endpoint)
}

func DevEndDeployment(cli runner.Client, moduleName string) error {
	return DevEndT(cli, moduleName, "deployment")
}

func DevEndStatefulSet(cli runner.Client, moduleName string) error {
	return DevEndT(cli, moduleName, "statefulset")
}

func DevEndT(cli runner.Client, moduleName string, moduleType string) error {
	cmd := cli.GetNhctl().Command(context.Background(), "dev", "end", "bookinfo", "-d", moduleName, "-t", moduleType)
	if stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(
		cli.SuiteName(), cmd, nil,
	); runner.Runner.CheckResult(
		cmd, stdout, stderr, err,
	) != nil {
		return err
	}
	//_ = k8sutils.WaitPod(
	//	cli.GetClientset(),
	//	cli.GetNhctl().Namespace,
	//	metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", moduleName).String()},
	//	func(i *v1.Pod) bool {
	//		return i.Status.Phase == v1.PodRunning && func() bool {
	//			for _, containerStatus := range i.Status.ContainerStatuses {
	//				if containerStatus.Ready {
	//					return true
	//				}
	//			}
	//			return false
	//		}()
	//	},
	//	time.Minute*5,
	//)
	return nil
}
