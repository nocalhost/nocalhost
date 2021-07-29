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
