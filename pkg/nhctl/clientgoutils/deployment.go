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
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func (c *ClientGoUtils) ListDeployments() ([]v1.Deployment, error) {
	deps, err := c.GetDeploymentClient().List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return deps.Items, nil
}

// Update deployment
// If wait, UpdateDeployment will not return until:
// 1. Deployment is ready
// 2. Previous revision of ReplicaSet terminated
// 3. Latest revision of ReplicaSet is ready
// After update, UpdateDeployment will clean up previous revision's events
// If Latest revision of ReplicaSet fails to be ready, return err
func (c *ClientGoUtils) UpdateDeployment(deployment *v1.Deployment, wait bool) (*v1.Deployment, error) {
	// Get current revision of replica set
	rss, err := c.GetSortedReplicaSetsByDeployment(deployment.Name)
	if err != nil {
		return nil, err
	}

	dep, err := c.GetDeploymentClient().Update(c.ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if !wait {
		return dep, nil
	}

	// Wait for deployment to be ready
	ready, _ := isDeploymentReady(dep)
	if !ready {
		err = c.WaitDeploymentToBeReady(dep.Name)
		if err != nil {
			return nil, err
		}
	}

	// Delete previous revision ReplicaSet's event
	if len(rss) == 0 { // No event needs to delete
		return dep, err
	}

	rsName := rss[len(rss)-1].Name
	events, err := c.ListEventsByReplicaSet(rsName)
	if err != nil {
		log.WarnE(err, fmt.Sprintf("Failed to delete events of %s", rsName))
		return dep, nil
	} else {
		log.Debugf("Clean up events of %s", rsName)
	}

	for _, event := range events {
		err = c.DeleteEvent(event.Name)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Failed to delete event %s", event.Name))
		} else {
			log.Logf("Event %s deleted", event.Name)
		}
	}

	err = c.WaitLatestRevisionReady(dep.Name)
	if err != nil {
		return nil, err
	}

	return dep, nil
}

func CheckIfDeploymentIsReplicaFailure(deploy *v1.Deployment) (bool, string, string, error) {
	if deploy == nil {
		return false, "", "", errors.New("failed to check a nil deployment")
	}

	for _, condition := range deploy.Status.Conditions {
		if condition.Type == v1.DeploymentReplicaFailure {
			return true, condition.Reason, condition.Message, nil
		}

	}
	return false, "", "", nil
}

func (c *ClientGoUtils) GetDeployment(name string) (*v1.Deployment, error) {
	dep, err := c.GetDeploymentClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateDeployment(deploy *v1.Deployment) (*v1.Deployment, error) {
	dep, err := c.ClientSet.AppsV1().Deployments(c.namespace).Create(c.ctx, deploy, metav1.CreateOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) DeleteDeployment(name string, wait bool) error {
	dep, err := c.ClientSet.AppsV1().Deployments(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = c.ClientSet.AppsV1().Deployments(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	labelMap := dep.Spec.Selector.MatchLabels

	if wait {
		log.Infof(
			"Waiting pods of %s to be terminated, this may take several minutes, depending on the load of your k8s cluster",
			name,
		)
		terminated := false
		for i := 0; i < 200; i++ {
			list, err := c.ListPodsByLabels(labelMap)
			utils.Should(err)
			if len(list) == 0 {
				log.Infof("All pods of %s have been terminated", name)
				terminated = true
				break
			}
			time.Sleep(2 * time.Second)
		}
		if !terminated {
			log.Warnf("Waiting pods of %s to be terminated timeout", name)
		}
	}
	return nil
}
