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
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) UpdateReplicaSet(rs *v1.ReplicaSet) (*v1.ReplicaSet, error) {
	rs2, err := c.ClientSet.AppsV1().ReplicaSets(c.namespace).Update(c.ctx, rs, metav1.UpdateOptions{})
	return rs2, errors.Wrap(err, "")
}
