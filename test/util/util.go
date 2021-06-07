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
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	clientgowatch "k8s.io/client-go/tools/watch"
	"nocalhost/pkg/nhctl/log"
	"os"
	"runtime"
	"strings"
	"time"
)

func WaitResourceToBeStatus(g cache.Getter, namespace string, resource string, label string, checker func(interface{}) bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	watchlist := cache.NewFilteredListWatchFromClient(
		g,
		resource,
		namespace,
		func(options *metav1.ListOptions) { options.LabelSelector = label })

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
	conditionFunc := func(e watch.Event) (bool, error) { return checker(e.Object), nil }
	obj, err := getRuntimeObjectByResource(resource)
	if err != nil {
		return false
	}
	event, err := clientgowatch.UntilWithSync(ctx, watchlist, obj, preConditionFunc, conditionFunc)
	if err != nil {
		log.Infof("wait pod has the label: %s to ready failed, error: %v, event: %v", label, err, event)
		return false
	}
	return true
}

// todo how to make it more elegant
func getRuntimeObjectByResource(resource string) (k8sruntime.Object, error) {
	switch resource {
	case "pods":
		return &v1.Pod{}, nil
	case "deployments":
		return &appsv1.Deployment{}, nil
	case "statefulsets":
		return &appsv1.StatefulSet{}, nil
	default:
		return nil, errors.New("not support resouce type: " + resource)
	}
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
	debug := os.Getenv(Local)
	if debug != "" {
		return false
	}
	if strings.Contains(runtime.GOOS, "darwin") {
		return true
	} else if strings.Contains(runtime.GOOS, "windows") {
		return true
	} else {
		return false
	}
}

func GetKubeconfig() string {
	kubeconfig := os.Getenv(KubeconfigPath)
	if kubeconfig == "" {
		kubeconfig = "/root/.kube/config"
	}
	return kubeconfig
}
