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
	"encoding/base64"
	"fmt"
	"testing"
)

func TestClientGoUtils_CreatePVC(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	if err != nil {
		panic(err)
	}

	dirBase64 := base64.StdEncoding.EncodeToString([]byte("/var/tmp/tmp"))
	labels := map[string]string{
		"nocalhost.dev/app":     "app01",
		"nocalhost.dev/service": "details",
		"nocalhost.dev/dir":     dirBase64,
	}
	annotations := map[string]string{"nocalhost.dev/dir": "/var/tmp/tmp"}

	pvc, err := client.CreatePVC("test01", labels, annotations, "10Gi", nil)
	if err != nil {
		fmt.Printf("%+v", err)
		panic(err)
	}
	fmt.Printf("%+v\n", pvc)
}

func TestClientGoUtils_GetPvcByLabels(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	if err != nil {
		panic(err)
	}

	dirBase64 := base64.StdEncoding.EncodeToString([]byte("/var/tmp/tmp"))
	//labels := map[string]string{"nocalhost.dev/app": "app", "nocalhost.dev/service": "details", "nocalhost.dev/dir": dirBase64}
	labels := map[string]string{"nocalhost.dev/service": "details1", "nocalhost.dev/dir": dirBase64}
	pvcs, err := client.GetPvcByLabels(labels)
	if err != nil {
		panic(err)
	}
	for _, pvc := range pvcs {
		fmt.Println(pvc.Name)
	}
}

func TestClientGoUtils_DeletePVC(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	if err != nil {
		panic(err)
	}

	err = client.DeletePVC("test01")
	if err != nil {
		fmt.Printf("%+v", err)
		panic(err)
	}
	fmt.Printf("pvc deleted\n")

}
