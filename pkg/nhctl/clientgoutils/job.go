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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *ClientGoUtils) CreateJob(job *batchv1.Job) (*batchv1.Job, error) {
	job, err := c.ClientSet.BatchV1().Jobs(c.namespace).Create(c.ctx, job, metav1.CreateOptions{})
	return job, errors.Wrap(err, "")
}

func (c *ClientGoUtils) DeleteJob(name string) error {
	propagationPolicy := metav1.DeletePropagationBackground
	return errors.Wrap(c.ClientSet.BatchV1().Jobs(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy}), "")
}

// This method can not list pods whose deployment is already deleted.
func (c *ClientGoUtils) ListPodsByJob(name string) (*corev1.PodList, error) {
	job, err := c.ClientSet.BatchV1().Jobs(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(job.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods, nil
}
