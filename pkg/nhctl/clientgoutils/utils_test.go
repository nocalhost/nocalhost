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
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewClientGoUtils(t *testing.T) {
	client := NewClientGoUtils("") // /Users/xinxinhuang/.kube

	depList, err := client.ClientSet.AppsV1().Deployments("xinxinhuang").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		printlnErr("err", err)
		return
	}

	for _, v := range depList.Items {
		fmt.Println(v.Name)
	}
}

func TestClientGoUtils_Create(t *testing.T) {
	client := NewClientGoUtils("/Users/xinxinhuang/.kube/admin-config")
	err := client.Create("/Users/xinxinhuang/WorkSpaces/helm/bookinfo-manifest/deployment/templates/pre-install/print-num-job-01.yaml", "demo3", true)
	if err != nil {
		printlnErr("failed : ", err)
	}
}

//func TestClientGoUtils_GetRestClient(t *testing.T) {
//	client := NewClientGoUtils("/Users/xinxinhuang/.kube/admin-config")
//	restClient , err := client.GetRestClient(BatchV1)
//	if err != nil {
//		printlnErr("failed !!!" , err)
//		return
//	}
//	fmt.Println(restClient)
//}
