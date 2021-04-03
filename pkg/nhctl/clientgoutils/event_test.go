/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

func TestKustomize(t *testing.T){
	fSys := fs.MakeRealFS()
	var out bytes.Buffer
	err := kustomize.RunKustomizeBuild(&out, fSys, "/Users/anur/Downloads/kustomize-demo/deploy/demo/base")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(out.Bytes()))
}

func TestClientGoUtils_ListEventsByReplicaSet(t *testing.T) {
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
