/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"bytes"
	"fmt"
	"k8s.io/cli-runtime/pkg/kustomize"
	"sigs.k8s.io/kustomize/pkg/fs"
	"testing"
)

var kubeConfigForTest = "~/kubeconfigs/c3bbeccc-b61a-411a-af39-3d07bfe91017"
var namespaceForTest = "nh6ihig"

func TestKustomize(t *testing.T) {
	fSys := fs.MakeRealFS()
	var out bytes.Buffer
	err := kustomize.RunKustomizeBuild(&out, fSys, "/Users/anur/Downloads/kustomize-demo/deploy/demo/base")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(out.Bytes()))
}

func TestListEventsByRS(t *testing.T) {
	client, _ := NewClientGoUtils("", "nh6ihig")
	list, err := client.ListEventsByReplicaSet("details-59c787d477")
	if err != nil {

		panic(err)
	}
	for _, event := range list {
		fmt.Printf("%s %s %s %s\n", event.Name, event.Reason, event.LastTimestamp.String(), event.Message)
	}
}

func TestClientGoUtils_DeleteEvent(t *testing.T) {
	client, _ := NewClientGoUtils(kubeConfigForTest, namespaceForTest)
	events, err := client.ListEventsByReplicaSet("details-59c787d477")
	if err != nil {
		panic(err)
	}
	for _, event := range events {
		err := client.DeleteEvent(event.Name)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s deleted\n", event.Name)
	}
}
