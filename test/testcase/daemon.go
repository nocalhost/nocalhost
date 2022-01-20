/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package testcase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"time"
)

func RestartDaemon(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "daemon", "restart")
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func StopDaemon(nhctl *runner.CLI) error {
	cmd := nhctl.Command(context.Background(), "daemon", "stop")
	return runner.Runner.RunWithCheckResult(nhctl.SuitName(), cmd)
}

func Exec(client runner.Client) error {
	_ = k8sutils.WaitPod(
		client.GetClientset(),
		client.GetNhctl().Namespace,
		metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "reviews").String()},
		func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
		time.Minute*30,
	)
	cmd := client.GetNhctl().Command(context.Background(), "exec", "bookinfo", "-d", "reviews", "-c", "ls")
	return runner.Runner.RunWithCheckResult(client.SuiteName(), cmd)
}

func PortForwardStart(nhctl runner.Client, module string, port int) error {
	return PortForwardStartT(nhctl, module, "deployment", port)
}

func PortForwardStartT(nhctl runner.Client, module, moduleType string, port int) error {
	podName, _ := nhctl.GetNhctl().Command(
		context.Background(), "dev", "pod", "bookinfo", "-d", module, "-t", moduleType).Output()

	//pods, err := nhctl.GetClientset().CoreV1().Pods(nhctl.GetNhctl().Namespace).List(
	//	context.Background(), metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", module).String()})
	//if err != nil {
	//	return errors.Wrap(err, "List pods error")
	//}
	//if pods == nil || len(pods.Items) < 1 {
	//	return errors.Errorf("Not found pods of module %v", module)
	//}
	//var name string
	//for _, pod := range pods.Items {
	//	if v1.PodRunning == pod.Status.Phase && pod.DeletionTimestamp == nil {
	//		name = pod.Name
	//		break
	//	}
	//}
	//if name == "" {
	//	return errors.New("pods status is not running")
	//}
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "port-forward",
		"start",
		"bookinfo",
		"-d",
		module,
		"-t",
		moduleType,
		"--pod",
		string(podName),
		fmt.Sprintf("-p%d:9080", port),
	)
	_, _, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return err
}

func PortForwardServiceStart(cli runner.Client, module string, port int) error {
	service, err := cli.GetClientset().CoreV1().
		Services(cli.GetNhctl().Namespace).
		Get(context.Background(), module, metav1.GetOptions{})
	if err != nil || service == nil {
		return errors.Errorf("service %s not found", module)
	}
	cmd := cli.GetKubectl().Command(
		context.Background(), "port-forward",
		"service/"+module,
		fmt.Sprintf("%d:9080", port),
	)
	return runner.Runner.RunWithCheckResult(cli.SuiteName(), cmd)
}

func StatusCheckPortForward(nhctl runner.Client, moduleName, moduleType string, port int) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "describe", "bookinfo", "-d", moduleName, "-t", moduleType)
	stdout, stderr, err := runner.Runner.Run(nhctl.SuiteName(), cmd)
	if err != nil {
		return errors.Errorf(
			"exec command: %v, error: %v, stdout: %s, stderr: %s", cmd.Args, err, stdout, stderr,
		)
	}
	service := profile2.SvcProfileV2{}
	_ = yaml.Unmarshal([]byte(stdout), &service)
	bytes, _ := json.Marshal(service)
	log.TestLogger(nhctl.SuiteName()).Info(string(bytes))
	if !service.PortForwarded {
		return errors.New("test case failed, should be port forwarding")
	}
	for _, e := range service.DevPortForwardList {
		if e.LocalPort == port && (e.Status != "LISTEN" && e.Status != "New") {
			return errors.Errorf("status: %s is not correct", e.Status)
		}
	}
	return nil
}

func PortForwardEnd(nhctl runner.Client, module string, port int) error {
	return PortForwardEndT(nhctl, module, "deployment", port)
}
func PortForwardEndT(nhctl runner.Client, module, moduleType string, port int) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "port-forward",
		"end",
		"bookinfo",
		"-d",
		module,
		"-t",
		moduleType,
		fmt.Sprintf("-p%d:9080", port),
	)
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}
