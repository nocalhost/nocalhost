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
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/util"
)

func RestartDaemon(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "daemon", "restart")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func StopDaemon(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(context.Background(), "daemon", "stop")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func Exec(nhctl *nhctlcli.CLI) error {
	util.WaitResourceToBeStatus(nhctl.Namespace, "pods", "app=reviews", func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
	cmd := nhctl.Command(context.Background(), "exec", "bookinfo", "-d", "reviews", "-c", "ls")
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func PortForwardStart(nhctl *nhctlcli.CLI, module string, port int) error {
	pods, err := util.Client.ClientSet.CoreV1().
		Pods(nhctl.Namespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: "app=" + module})
	if err != nil {
		return errors.Wrap(err, "List pods error")
	}
	if pods == nil || len(pods.Items) < 1 {
		return errors.Errorf("Not found pods of module %v", module)
	}
	var name string
	for _, pod := range pods.Items {
		if v1.PodRunning == pod.Status.Phase {
			name = pod.Name
			break
		}
	}
	if name == "" {
		return errors.New("pods status is not running")
	}
	cmd := nhctl.Command(context.Background(), "port-forward",
		"start",
		"bookinfo",
		"-d",
		module,
		"--pod",
		name,
		fmt.Sprintf("-p%d:9080", port))
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func PortForwardServiceStart(cli *nhctlcli.CLI, module string, port int) error {
	service, err := util.Client.ClientSet.CoreV1().
		Services(cli.Namespace).
		Get(context.Background(), module, metav1.GetOptions{})
	if err != nil || service == nil {
		return errors.Errorf("service %s not found", module)
	}
	cmd := cli.Command(context.Background(), "port-forward",
		"service/"+module,
		fmt.Sprintf("%d:9080", port))
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}

func StatusCheckPortForward(nhctl *nhctlcli.CLI, moduleName string, port int) error {
	cmd := nhctl.Command(context.Background(), "describe", "bookinfo", "-d", moduleName)
	stdout, stderr, err := nhctlcli.Runner.Run(cmd)
	if err != nil {
		return errors.Errorf("exec command: %v, error: %v, stdout: %s, stderr: %s",
			cmd.Args, err, stdout, stderr)
	}
	service := profile2.SvcProfileV2{}
	_ = yaml.Unmarshal([]byte(stdout), &service)
	bytes, _ := json.Marshal(service)
	log.Info(string(bytes))
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

func PortForwardEnd(nhctl *nhctlcli.CLI, module string, port int) error {
	cmd := nhctl.Command(context.Background(), "port-forward",
		"end",
		"bookinfo",
		"-d",
		module,
		fmt.Sprintf("-p%d:9080", port))
	return nhctlcli.Runner.RunWithCheckResult(cmd)
}
