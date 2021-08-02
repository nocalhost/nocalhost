/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/pkg/nhctl/log"
	"os"
	"regexp"
	"testing"
)

//
func getClient() *ClientGoUtils {
	client, err := NewClientGoUtils("", "nh6xury")
	if err != nil {
		panic(err)
	}
	return client
}

func TestClientGoUtils_GetDeployment(t *testing.T) {
	d, err := getClient().GetDeployment("productpage1")
	if err != nil {
		panic(err)
	}
	fmt.Println(d.Name)
}

func TestPortForwardNotFound(t *testing.T) {
	utils, err := NewClientGoUtils(clientcmd.RecommendedHomeFile, "test")
	if err != nil {
		log.Fatal(err)
	}
	err = utils.PortForwardAPod(PortForwardAPodRequest{
		Listen: []string{"0.0.0.0"},
		Pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "asdf",
				Namespace: "test",
			},
		},
		LocalPort: 2222,
		PodPort:   2222,
		Streams:   genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		StopCh:    make(chan struct{}),
		ReadyCh:   make(chan struct{}),
	})

	if err == nil {
		fmt.Println("failed")
		return
	}

	if found, _ := regexp.Match("pods \"(.*?)\" not found", []byte(err.Error())); found {
		fmt.Println("ok")
	} else {
		fmt.Println("not ok")
	}
}
