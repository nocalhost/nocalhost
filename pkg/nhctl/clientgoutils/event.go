/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
