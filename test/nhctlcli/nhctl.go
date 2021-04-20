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

package nhctlcli

func NewNhctl(kubeconfig, namespace string) *CLI {
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
