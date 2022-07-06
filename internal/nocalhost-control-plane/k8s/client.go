/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package k8s

import (
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	RestConfig *rest.Config
	ClientSet  kubernetes.Interface
	Namespace  string
)

func init() {
	var err error
	RestConfig, err = config.GetConfig()
	if err != nil {
		panic(err)
	}
	ClientSet, err = kubernetes.NewForConfig(RestConfig)
	if err != nil {
		panic(err)
	}
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		panic(err)
	}
	Namespace = string(ns)
}
