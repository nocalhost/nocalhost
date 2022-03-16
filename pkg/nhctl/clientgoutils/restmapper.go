/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
	"nocalhost/pkg/nhctl/log"
	"strings"
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

func (c *ClientGoUtils) ResourceFor(resourceArg string, tryLoadFromCache bool) schema.GroupVersionResource {
	c.gvrCacheLock.Lock()
	if c.gvrCache == nil {
		c.gvrCache = map[string]schema.GroupVersionResource{}
	}

	if resource, ok := c.gvrCache[resourceArg]; ok && tryLoadFromCache {
		c.gvrCacheLock.Unlock()
		return resource
	}
	c.gvrCacheLock.Unlock()

	if resourceArg == "*" {
		return schema.GroupVersionResource{Resource: resourceArg}
	}

	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(strings.ToLower(resourceArg))
	gvr := schema.GroupVersionResource{}
	if fullySpecifiedGVR != nil {
		gvr, _ = c.restMapper.ResourceFor(*fullySpecifiedGVR)
	}
	if gvr.Empty() {
		var err error
		gvr, err = c.restMapper.ResourceFor(groupResource.WithVersion(""))
		if err != nil {
			if !nonStandardResourceNames.Has(groupResource.String()) {
				if len(groupResource.Group) == 0 {
					log.Logf("Warning: the server doesn't have a resource type '%s'\n", groupResource.Resource)
				} else {
					log.Logf(
						"Warning: the server doesn't have a resource type '%s' in group '%s'\n", groupResource.Resource,
						groupResource.Group,
					)
				}
			}
			return schema.GroupVersionResource{Resource: resourceArg}
		}
	}

	c.gvrCacheLock.Lock()
	c.gvrCache[resourceArg] = gvr
	c.gvrCacheLock.Unlock()
	return gvr
}

func (c *ClientGoUtils) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return c.restMapper.KindFor(resource)
}

func (c *ClientGoUtils) ResourceForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, bool, error) {
	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}

	nsed, err := IsNamespaced(mapping.Resource, c.restMapper)
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}

	return mapping.Resource, nsed, nil
}

func IsNamespaced(gvr schema.GroupVersionResource, mapper meta.RESTMapper) (bool, error) {
	kind, err := mapper.KindFor(gvr)
	if err != nil {
		return false, err
	}

	mapping, err := mapper.RESTMapping(kind.GroupKind(), kind.Version)
	if err != nil {
		return false, err
	}

	return mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}
