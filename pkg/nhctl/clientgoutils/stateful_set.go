/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"nocalhost/pkg/nhctl/log"
	"time"
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

func (c *ClientGoUtils) CreateStatefulSetAndWait(s *v1.StatefulSet) (*v1.StatefulSet, error) {
	ss, err := c.CreateStatefulSet(s)
	if err != nil {
		return nil, err
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

func (c *ClientGoUtils) ScaleStatefulSetReplicasToOne(name string) error {
	scale, err := c.GetStatefulSetClient().GetScale(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 || scale.Spec.Replicas == 0 {
		scale.Spec.Replicas = 1
		_, err = c.GetStatefulSetClient().UpdateScale(c.ctx, name, scale, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Info("Waiting replicas scale to 1, it may take several minutes...")
		for i := 0; i < 300; i++ {
			time.Sleep(1 * time.Second)
			ss, err := c.GetStatefulSet(name)
			if err != nil {
				return errors.Wrap(err, "")
			}
			if ss.Status.ReadyReplicas == 1 && ss.Status.Replicas == 1 {
				log.Info("Replicas has been scaled to 1")
				return nil
			}
		}
		return errors.New("Waiting replicas scaling to 1 timeout")
	} else {
		log.Info("Replicas has already been scaled to 1")
	}
	return nil
}

func (c *ClientGoUtils) ListStatefulSets() ([]v1.StatefulSet, error) {
	ops := metav1.ListOptions{}
	if len(c.labels) > 0 {
		ops.LabelSelector = labels.Set(c.labels).String()
	}
	deps, err := c.GetStatefulSetClient().List(c.ctx, ops)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return deps.Items, nil
}
