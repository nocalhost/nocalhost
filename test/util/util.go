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

package util

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	clientgowatch "k8s.io/client-go/tools/watch"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/test/nhctlcli"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var Client *clientgoutils.ClientGoUtils

func Init(cli *nhctlcli.CLI) {
	temp, err := clientgoutils.NewClientGoUtils(cli.KubeConfig, cli.Namespace)
	if err != nil {
		panic(fmt.Sprintf("init k8s client error: %v", err))
	}
	Client = temp
}

func WaitForCommandDone(command string, args ...string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err.Error()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return false, "Command timeout"
	}
	return cmd.ProcessState.Success(), string(output)
}

func WaitToBeStatus(namespace string, resource string, label string, checker func(interface{}) bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	watchlist := cache.NewFilteredListWatchFromClient(
		Client.ClientSet.CoreV1().RESTClient(),
		resource,
		namespace,
		func(options *metav1.ListOptions) {
			options.LabelSelector = label
		})

	preConditionFunc := func(store cache.Store) (bool, error) {
		if len(store.List()) == 0 {
			return false, nil
		}
		for _, p := range store.List() {
			if !checker(p) {
				return false, nil
			}
		}
		return true, nil
	}

	conditionFunc := func(e watch.Event) (bool, error) {
		return checker(e.Object), nil
	}
	event, err := clientgowatch.UntilWithSync(ctx, watchlist, &v1.Pod{}, preConditionFunc, conditionFunc)
	if err != nil {
		fmt.Printf("wait pod has the label: %s to ready failed, error: %v, event: %v\n", label, err, event)
		return false
	}
	return true
}

func TimeoutChecker(d time.Duration, cancanFunc func()) {
	tick := time.Tick(d)
	for {
		select {
		case <-tick:
			if cancanFunc != nil {
				cancanFunc()
			}
			panic(fmt.Sprintf("test case failed, timeout: %v", d))
		}
	}
}

func NeedsToInitK8sOnTke() bool {
	if strings.Contains(runtime.GOOS, "darwin") {
		return true
	} else if strings.Contains(runtime.GOOS, "windows") {
		return true
	} else {
		return false
	}
}

func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG_PATH")
	if kubeconfig == "" {
		kubeconfig = "/root/.kube/config"
	}
	return kubeconfig
}
