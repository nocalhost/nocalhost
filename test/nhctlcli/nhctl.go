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

package nhctlcli

import "os"

func NewNhctl(namespace string) *CLI {
	kubeconfig := os.Getenv("KUBECONFIG_PATH")
	if kubeconfig == "" {
		kubeconfig = "/root/.kube/config"
	}
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
	}
	return NewCLI(c, namespace)
}

type Conf struct {
	kubeconfig string
	namespace  string
}

func (c *Conf) GetKubeConfig() string {
	return c.kubeconfig
}
func (c *Conf) GetNamespace() string {
	return c.namespace
}
