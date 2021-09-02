/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func (c *ClientGoUtils) CheckExistNameSpace(name string) error {
	_, err := c.ClientSet.CoreV1().Namespaces().Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (c *ClientGoUtils) CreateNameSpace(name string) error {
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: c.labels}}
	_, err := c.ClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	return errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateNamespaceINE(ns string) error {
	if err := c.CreateNameSpace(ns); err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "")
	}
	return nil
}

func (c *ClientGoUtils) DeleteNameSpace(name string, wait bool) error {
	err := c.ClientSet.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if wait {
		timeout := time.After(5 * time.Minute)
		tick := time.Tick(200 * time.Millisecond)
		for {
			select {
			case <-timeout:
				return errors.New("timeout with 5 minute")
			case <-tick:
				err := c.CheckExistNameSpace(name)
				if err != nil {
					return nil
				}
			}
		}
	}
	return err
}
