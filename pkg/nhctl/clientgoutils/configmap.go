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
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) DeleteConfigMapByName(name string) error {
	return errors.Wrap(c.ClientSet.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) GetConfigMaps() ([]v1.ConfigMap, error) {
	var result []v1.ConfigMap
	list, err := c.ClientSet.CoreV1().ConfigMaps(c.namespace).List(c.ctx, metav1.ListOptions{})
	if list != nil {
		result = list.Items
	}
	return result, errors.Wrap(err, "")
}
