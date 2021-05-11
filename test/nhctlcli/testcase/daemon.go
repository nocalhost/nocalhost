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
	"fmt"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/util"
)

func RestartDaemon(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "daemon", "restart")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func StopDaemon(nhctl *nhctlcli.CLI) {
	cmd := nhctl.Command(context.Background(), "daemon", "stop")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func Exec(nhctl *nhctlcli.CLI) {
	util.WaitResourceToBeStatus(nhctl.Namespace, "pods", "app=reviews", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	cmd := nhctl.Command(context.Background(), "exec", "bookinfo", "-d", "reviews", "-c", "ls")
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func PortForwardStart(nhctl *nhctlcli.CLI, module string, port int) {
	pods, err := util.Client.ClientSet.CoreV1().
		Pods(nhctl.Namespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: "app=" + module})
	if err != nil {
		panic(fmt.Sprintf("List pods error: %v", err))
	}
	if pods == nil || len(pods.Items) < 1 {
		panic(fmt.Sprintf("Not found pods of module %v", module))
	}
	cmd := nhctl.Command(context.Background(), "port-forward",
		"start",
		"bookinfo",
		"-d",
		module,
		"--pod",
		pods.Items[0].Name,
		fmt.Sprintf("-p%d:9080", port))
	nhctlcli.Runner.RunPanicIfError(cmd)
}

func StatusCheckPortForward(nhctl *nhctlcli.CLI, moduleName string, port int) {
	cmd := nhctl.Command(context.Background(), "describe", "bookinfo", "-d", moduleName)
	stdout, stderr, err := nhctlcli.Runner.Run(cmd)
	if err != nil {
		panic(fmt.Sprintf(
			"exec command: %s, error: %v, stdout: %s, stderr: %s\n", cmd.Args, err, stdout, stderr))
	}
	service := profile2.SvcProfileV2{}
	_ = yaml.Unmarshal([]byte(stdout), &service)
	fmt.Println(service)
	if !service.PortForwarded {
		panic("test case failed, should be port forwarding")
	}
	for _, e := range service.DevPortForwardList {
		if e.LocalPort == port && (e.Status != "LISTEN" && e.Status != "New") {
			panic(fmt.Sprintf("status: %s is not correct", e.Status))
		}
	}
}

func PortForwardEnd(nhctl *nhctlcli.CLI, module string, port int) {
	cmd := nhctl.Command(context.Background(), "port-forward",
		"end",
		"bookinfo",
		"-d",
		module,
		fmt.Sprintf("-p%d:9080", port))
	nhctlcli.Runner.RunPanicIfError(cmd)
}
