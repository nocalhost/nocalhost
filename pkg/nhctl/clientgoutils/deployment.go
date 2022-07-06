/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func (c *ClientGoUtils) ListDeployments() ([]v1.Deployment, error) {
	ops := c.getListOptions()
	deps, err := c.GetDeploymentClient().List(c.ctx, ops)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]v1.Deployment, 0)
	if !c.includeDeletedResources {
		for _, d := range deps.Items {
			if d.DeletionTimestamp == nil {
				result = append(result, d)
			}
		}
	} else {
		result = deps.Items
	}
	return result, nil
}

// UpdateDeployment Update deployment
// If wait, UpdateDeployment will not return until:
// 1. Deployment is ready
// 2. Previous revision of ReplicaSet terminated
// 3. Latest revision of ReplicaSet is ready
// After update, UpdateDeployment will clean up previous revision's events
// If Latest revision of ReplicaSet fails to be ready, return err
func (c *ClientGoUtils) UpdateDeployment(deployment *v1.Deployment, wait bool) (*v1.Deployment, error) {

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

	err = c.WaitLatestRevisionReady(dep.Name)
	if err != nil {
		return nil, err
	}

	return dep, nil
}

func (c *ClientGoUtils) CreateDeploymentAndWait(deployment *v1.Deployment) (*v1.Deployment, error) {

	dep, err := c.GetDeploymentClient().Create(c.ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	// Wait for deployment to be ready
	ready, _ := isDeploymentReady(dep)
	if !ready {
		err = c.WaitDeploymentToBeReady(dep.Name)
		if err != nil {
			return nil, err
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

func (c *ClientGoUtils) ScaleDeploymentReplicasToOne(name string) error {
	deploymentsClient := c.GetDeploymentClient()
	scale, err := deploymentsClient.GetScale(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 || scale.Spec.Replicas == 0 {
		log.Infof("Deployment %s replicas is %d now, scaling it to 1", name, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(c.ctx, name, scale, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "")
		} else {
			log.Info("Replicas has been set to 1")
		}
	} else {
		log.Infof("Deployment %s replicas is already 1, no need to scale", name)
	}
	return nil
}
