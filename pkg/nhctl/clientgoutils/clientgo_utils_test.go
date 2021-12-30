/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"encoding/json"
	"fmt"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

//
func getClient() *ClientGoUtils {
	client, err := NewClientGoUtils("", "")
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

//func TestPortForwardNotFound(t *testing.T) {
//	utils, err := NewClientGoUtils(clientcmd.RecommendedHomeFile, "test")
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = utils.PortForwardAPod(PortForwardAPodRequest{
//		Listen: []string{"0.0.0.0"},
//		Pod: corev1.Pod{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      "asdf",
//				Namespace: "test",
//			},
//		},
//		LocalPort: 2222,
//		PodPort:   2222,
//		Streams:   genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
//		StopCh:    make(chan struct{}),
//		ReadyCh:   make(chan struct{}),
//	})
//
//	if err == nil {
//		fmt.Println("failed")
//		return
//	}
//
//	if found, _ := regexp.Match("pods \"(.*?)\" not found", []byte(err.Error())); found {
//		fmt.Println("ok")
//	} else {
//		fmt.Println("not ok")
//	}
//}

func TestClientGoUtils_ListHPA(t *testing.T) {
	c, err := NewClientGoUtils("", "nocalhost-test")
	if err != nil {
		panic(err)
	}
	hs, err := c.ListHPA()
	if err != nil {
		panic(err)
	}
	for _, h := range hs {
		fmt.Println(h.Name)
	}
}

func TestClientGoUtils_ListResourceInfo(t *testing.T) {
	client := getClient()
	//crds, err := client.ListResourceInfo("crd")
	//if err != nil {
	//	return
	//}
	////for _, crd := range crds {
	////fmt.Printf("%v\n", crds[0].Object)
	//////fmt.Printf("%s %s %s\n", cc.Name, cc.Kind, cc.APIVersion)
	//////}
	//err := client.GetInformer("deployments.v1.apps")
	//if err != nil {
	//	panic(err)
	//}

	gr, err := client.GetAPIGroupResources()
	if err != nil {
		panic(err)
	}
	fmt.Println(gr)

	crds, err := client.ListResourceInfo("all")
	if err != nil {
		return
	}
	crd := crds[0]
	um, ok := crd.Object.(*unstructured.Unstructured)
	if !ok {
		panic(err)
	}
	jsonBytes, err := um.MarshalJSON()
	if err != nil {
		panic(err)
	}
	crdObj := &apiextensions.CustomResourceDefinition{}
	err = json.Unmarshal(jsonBytes, crdObj)
	if err != nil {
		panic(err)
	}
	fmt.Println(crds)
}
