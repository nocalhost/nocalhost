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

package k8sutils

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	toolswatch "k8s.io/client-go/tools/watch"
	"k8s.io/kubectl/pkg/scheme"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func ValidateDNS1123Name(name string) bool {
	errs := validation.IsDNS1123Subdomain(name)
	if len(errs) == 0 {
		return true
	} else {
		return false
	}
}

func WaitPod(client *kubernetes.Clientset, namespace string, listOptions v1.ListOptions,
	checker func(*corev1.Pod) bool, timeout time.Duration) error {
	return WaitResource(
		client,
		client.CoreV1().RESTClient(),
		namespace,
		corev1.SchemeGroupVersion.WithKind("Pod"),
		listOptions,
		func(i interface{}) bool { return checker(i.(*corev1.Pod)) },
		timeout,
	)
}

func WaitDeployment(client *kubernetes.Clientset, namespace string, listOptions v1.ListOptions,
	checker func(*appsv1.Deployment) bool, timeout time.Duration) error {
	return WaitResource(
		client,
		client.AppsV1().RESTClient(),
		namespace,
		appsv1.SchemeGroupVersion.WithKind("Deployment"),
		listOptions,
		func(i interface{}) bool { return checker(i.(*appsv1.Deployment)) },
		timeout,
	)
}

func WaitResource(client *kubernetes.Clientset, g cache.Getter, namespace string, gvk schema.GroupVersionKind,
	listOptions v1.ListOptions, checker func(interface{}) bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	groupResources, _ := restmapper.GetAPIGroupResources(client)
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	restMapping, _ := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)

	watchlist := cache.NewFilteredListWatchFromClient(
		g,
		restMapping.Resource.Resource,
		namespace,
		func(options *v1.ListOptions) {
			options.LabelSelector = listOptions.LabelSelector
			options.FieldSelector = listOptions.FieldSelector
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

	object, err := scheme.Scheme.New(gvk)
	if err != nil {
		return err
	}

	conditionFunc := func(e watch.Event) (bool, error) { return checker(e.Object), nil }
	event, err := toolswatch.UntilWithSync(ctx, watchlist, object, preConditionFunc, conditionFunc)
	if err != nil {
		log.Infof("wait resource: %s to ready failed, error: %v, event: %v",
			restMapping.Resource.Resource, err, event)
		return err
	}
	return nil
}
