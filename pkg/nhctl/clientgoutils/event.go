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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/utils"
)

func (c *ClientGoUtils) ListEventsByReplicaSet(rsName string) ([]corev1.Event, error) {
	list, err := c.ClientSet.CoreV1().Events(c.namespace).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	results := make([]corev1.Event, 0)

	if list == nil {
		return results, nil
	}

	for _, event := range list.Items {
		if event.InvolvedObject.Kind == "ReplicaSet" && event.InvolvedObject.Name == rsName {
			results = append(results, event)
		}
	}
	return results, nil
}

func (c *ClientGoUtils) ListEvents() ([]corev1.Event, error) {
	list, err := c.ClientSet.CoreV1().Events(c.namespace).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	results := make([]corev1.Event, 0)

	if list == nil {
		return results, nil
	}
	return list.Items, nil
}

func (c *ClientGoUtils) ListEventsByStatefulSet(name string) ([]corev1.Event, error) {

	allEvents, err := c.ListEvents()
	if err != nil {
		return nil, err
	}

	results := make([]corev1.Event, 0)
	for _, event := range allEvents {
		if event.InvolvedObject.Kind == "StatefulSet" && event.InvolvedObject.Name == name {
			results = append(results, event)
		}
	}
	return results, nil
}

func (c *ClientGoUtils) DeleteEvent(name string) error {
	return errors.Wrap(c.ClientSet.CoreV1().Events(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) DeleteEvents(evs []corev1.Event, continueOnErr bool) error {
	for _, ev := range evs {
		err := c.DeleteEvent(ev.Name)
		if err != nil {
			if continueOnErr {
				utils.Should(err)
			} else {
				return err
			}
		}
	}
	return nil
}
