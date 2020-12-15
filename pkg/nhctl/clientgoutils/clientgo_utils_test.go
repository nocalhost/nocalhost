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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"nocalhost/internal/nhctl/utils"
)

func TestNewClientGoUtils(t *testing.T) {

}

func TestClientGoUtils_Delete(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	Must(err)
	err = client.Delete("/tmp/pre-install-cm.yaml", "unit-test")
	if err != nil {
		panic(err)
	}
}

func TestClientGoUtils_Create(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	if err != nil {
		panic(err)
	}
	err = client.Create("/tmp/pre-install-cm.yaml", "unit-test", false, true)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	//s := runtime.Scheme{}
	//s.New()
}
