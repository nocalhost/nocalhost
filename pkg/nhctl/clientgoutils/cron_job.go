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
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListPodsByCronJob This method can not list pods whose deployment is already deleted.
func (c *ClientGoUtils) ListPodsByCronJob(name string) (*corev1.PodList, error) {
	job, err := c.ClientSet.BatchV1beta1().CronJobs(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(job.Spec.JobTemplate.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods, nil
}

func (c *ClientGoUtils) UpdateCronJob(cj *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {
	cj, err := c.GetCronJobsClient().Update(c.ctx, cj, metav1.UpdateOptions{})
	return cj, errors.Wrap(err, "")
}
