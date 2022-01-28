/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"path/filepath"
	"testing"
)

func TestPod(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	join := filepath.Join(clientcmd.RecommendedConfigDir, "mesh")
	configFlags.KubeConfig = &join
	factory := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))
	set, _ := factory.KubernetesClientSet()
	namespace := "naison"
	controller := NewServiceHandler(factory, set, namespace, "productpage")
	zero, annotation, ports, s, err := controller.ScaleToZero()
	fmt.Println(zero, annotation, ports, s, err)
	err = restore(factory, set, namespace, s)
	fmt.Println(err)
}

func TestPortForward(t *testing.T) {
	path := filepath.Join(homedir.HomeDir(), ".kube", "minikube")
	ns := "anur"
	c := ConnectOptions{
		Ctx:            context.TODO(),
		KubeconfigPath: path,
		Namespace:      ns,
	}
	err := c.InitClient(context.Background())
	if err != nil {
		panic(err)
	}
	if err = c.portForward(context.TODO()); err != nil {
		panic(err)
	}
}
