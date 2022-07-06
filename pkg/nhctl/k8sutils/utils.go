/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package k8sutils

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimejson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	toolswatch "k8s.io/client-go/tools/watch"
	"k8s.io/kubectl/pkg/scheme"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/utils"
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

func WaitResource(client *kubernetes.Clientset, g cache.Getter, namespace string, gvk schema.GroupVersionKind,
	listOptions v1.ListOptions, checker func(interface{}) bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	groupResources, _ := restmapper.GetAPIGroupResources(client)
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	restMapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

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

// gen or get kubeconfig from local path by kubeconfig content
// we would gen or get kubeconfig file named by hash
func GetOrGenKubeConfigPath(kubeconfigContent string) string {
	dir := nocalhost_path.GetNhctlKubeconfigDir(utils.Sha1ToString(kubeconfigContent))
	path := fp.NewFilePath(dir)
	if path.ReadFile() != "" {
		return dir
	} else {
		_ = path.RelOrAbs("../").Mkdir()
		_ = path.WriteFile(kubeconfigContent)
		return dir
	}
}

func GetObjectMetaData(obj interface{}) *metadataOnlyObject {
	var caseSensitiveJsonIterator = runtimejson.CaseSensitiveJSONIterator()
	marshal, errs := caseSensitiveJsonIterator.Marshal(obj)
	if errs != nil {
		return nil
	}
	v := &metadataOnlyObject{}
	if errs = caseSensitiveJsonIterator.Unmarshal(marshal, v); errs != nil {
		return nil
	}
	return v
}

func GetNamespaceAndNameFromObjectMeta(obj interface{}) (namespace, name string, errs error) {
	v := GetObjectMetaData(obj)
	return v.Namespace, v.Name, nil
}

type metadataOnlyObject struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
}
