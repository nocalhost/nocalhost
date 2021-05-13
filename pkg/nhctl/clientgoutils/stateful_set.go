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
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) UpdateStatefulSet(statefulSet *v1.StatefulSet, wait bool) (*v1.StatefulSet, error) {

	ss, err := c.GetStatefulSetClient().Update(c.ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if !wait {
		return ss, nil
	}

	if ss.Status.ReadyReplicas == 1 {
		return ss, nil
	}
	log.Debug("StatefulSet has not been ready yet")

	if err = c.WaitStatefulSetToBeReady(ss.Name); err != nil {
		return nil, err
	}
	return ss, nil
}

func (c *ClientGoUtils) DeleteStatefulSet(name string) error {
	return errors.Wrap(c.ClientSet.AppsV1().StatefulSets(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) CreateStatefulSet(s *v1.StatefulSet) (*v1.StatefulSet, error) {
	ss, err := c.ClientSet.AppsV1().StatefulSets(c.namespace).Create(c.ctx, s, metav1.CreateOptions{})
	return ss, errors.Wrap(err, "")
}
