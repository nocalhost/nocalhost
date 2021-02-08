/*
Copyright 2021 The Nocalhost Authors.
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

package app

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestNocalHostAppConfig_GetSvcConfig(t *testing.T) {
	bytes, err := ioutil.ReadFile("/tmp/v2.yaml")
	if err != nil {
		panic(err)
	}

	config := NocalHostAppConfigV2{}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", config)
}

func TestConvert(t *testing.T) {

	bytes, err := ioutil.ReadFile("/tmp/bookinfo_v1.yaml")
	if err != nil {
		panic(err)
	}

	config := &NocalHostAppConfig{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		panic(err)
	}

	configV2, err := convertConfigV1ToV2(config)
	if err != nil {
		panic(err)
	}
	v2Bytes, err := yaml.Marshal(configV2)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("/tmp/bookinfo_v2.yaml", v2Bytes, DefaultNewFilePermission)
	if err != nil {
		panic(err)
	}
}
