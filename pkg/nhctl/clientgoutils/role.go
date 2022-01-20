/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) GetClusterRole(name string) (*v1.ClusterRole, error) {
	r, err := c.ClientSet.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	return r, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateAdminClusterRoleINE(name string) (*v1.ClusterRole, error) {
	if cr, err := c.CreateAdminClusterRole(name); err != nil && !k8serrors.IsAlreadyExists(err) {
		return cr, err
	} else {
		return cr, nil
	}
}

func (c *ClientGoUtils) CreateAdminClusterRole(name string) (*v1.ClusterRole, error) {
	rule := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"*"},
			Resources: []string{"*"},
			APIGroups: []string{"*"},
		},
	}
	return c.CreateClusterRole(name, rule)
}

func (c *ClientGoUtils) CreateClusterRole(name string, rule []rbacv1.PolicyRule) (*v1.ClusterRole, error) {
	role := &rbacv1.ClusterRole{}
	role.Name = name
	role.Rules = rule
	role.Labels = c.labels
	r, err := c.ClientSet.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	return r, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateRoleBindingWithClusterRole(name, clusterRoleName string) (*v1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{}
	rb.Name = name
	rb.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     clusterRoleName,
	}
	rb.Labels = c.labels
	rb, err := c.ClientSet.RbacV1().RoleBindings(c.namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
	return rb, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateRoleBindingWithClusterRoleINE(name, clusterRoleName string) (*v1.RoleBinding, error) {
	if rb, err := c.CreateRoleBindingWithClusterRole(name, clusterRoleName); err != nil && !k8serrors.IsAlreadyExists(err) {
		return rb, err
	} else {
		return rb, nil
	}
}

func (c *ClientGoUtils) AddClusterRoleToRoleBinding(roleBinding, clusterRole, serviceAccount string) error {
	rb, err := c.ClientSet.RbacV1().RoleBindings(c.namespace).Get(context.TODO(), roleBinding, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}
	rb.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     clusterRole,
	}

	// skip if present
	for _, subject := range rb.Subjects {
		if subject.Name == serviceAccount {
			return nil
		}
	}

	rb.Subjects = append(
		rb.Subjects, rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: c.namespace,
			Name:      serviceAccount,
		},
	)

	_, err = c.ClientSet.RbacV1().RoleBindings(c.namespace).Update(context.TODO(), rb, metav1.UpdateOptions{})
	return errors.Wrap(err, "")
}
