/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package k8sutils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	if ValidateDNS1123Name("-111-11") {
		fmt.Println("valid")
	} else {
		fmt.Println("invalid")
	}
}

func TestWaitPod(t *testing.T) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{},
	)
	config, _ := clientConfig.ClientConfig()

	clientSet, err := kubernetes.NewForConfig(config)
	err = WaitPod(
		clientSet,
		"test",
		v1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", "reviews-d86f7d4f5-jkr8g").String()},
		func(i *corev1.Pod) bool { return i.Status.Phase != corev1.PodRunning },
		time.Minute*30,
	)
	if err != nil {
		fmt.Println(err)
	}
}

func TestLabel(t *testing.T) {
	s := fields.OneTermNotEqualSelector("app", "test").String()
	fmt.Println(s)
}
