/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/restmapper"
)

func (c *ClientGoUtils) GetAPIGroupResources() ([]*restmapper.APIGroupResources, error) {
	gr, err := restmapper.GetAPIGroupResources(c.ClientSet)
	return gr, errors.WithStack(err)
}

// IsClusterAdmin judge weather is cluster scope kubeconfig or not
func (c *ClientGoUtils) IsClusterAdmin() bool {
	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: "*",
				Group:     "*",
				Verb:      "*",
				Name:      "*",
				Version:   "*",
				Resource:  "*",
			},
		},
	}

	response, err := c.ClientSet.AuthorizationV1().SelfSubjectAccessReviews().Create(
		context.TODO(), arg, metav1.CreateOptions{},
	)
	if err != nil || response == nil {
		return false
	}
	return response.Status.Allowed
}
