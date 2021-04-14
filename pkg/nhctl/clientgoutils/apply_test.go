/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package clientgoutils

import (
	"fmt"
	"testing"
)

func TestClientGoUtils_Apply(t *testing.T) {
	client := getClient()
	err := client.Apply(
		"/Users/xinxinhuang/.nh/nhctl/application/bookinfo-coding/resources/manifest/templates/ratings.yaml",
	)
	if err != nil {
		panic(err)
	}
}

func TestClientGoUtils_CreateResourceInfo(t *testing.T) {
	client := getClient()

	infos, err := client.GetResourceInfoFromFiles(
		[]string{
			"/Users/xinxinhuang/.nh/nhctl/application/bookinfo-coding/resources/manifest/templates/ratings.yaml",
		},
		true,
	)
	if err != nil {
		panic(err)
	}

	for _, info := range infos {
		fmt.Println(info.Object.GetObjectKind().GroupVersionKind().Kind)
		if info.Object.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
			//err = client.UpdateResourceInfoByClientSide(info)
			err = client.ApplyResourceInfo(info)
			if err != nil {
				panic(err)
			}
		}
	}
}

func TestClientGoUtils_UpdateResourceInfo(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	if err != nil {
		panic(err)
	}

	infos, err := client.GetResourceInfoFromFiles([]string{"/tmp/yaml/ubuntu.yaml"}, true)
	if err != nil {
		panic(err)
	}

	for _, info := range infos {
		err = client.UpdateResourceInfoByServerSide(info)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}
