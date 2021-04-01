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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// This method can not list pods whose deployment is already deleted.
func (c *ClientGoUtils) ListPodsByDeployment(name string) (*corev1.PodList, error) {
	deployment, err := c.ClientSet.AppsV1().Deployments(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(deployment.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods, nil
}

func (c *ClientGoUtils) ListPodsByStatefulSet(name string) (*corev1.PodList, error) {
	ss, err := c.ClientSet.AppsV1().StatefulSets(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(ss.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods, nil
}

func (c *ClientGoUtils) ListPodsByLabels(labelMap map[string]string) ([]corev1.Pod, error) {
	set := labels.Set(labelMap)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	result := make([]corev1.Pod, 0)
	for _, pod := range pods.Items {
		result = append(result, pod)
	}
	return result, nil
}
