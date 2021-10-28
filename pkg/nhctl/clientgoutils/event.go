/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"k8s.io/kubectl/pkg/scheme"
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

func (c *ClientGoUtils) SearchEvents(objOrRef runtime.Object) (*corev1.EventList, error) {
	client := c.ClientSet.CoreV1()
	ref, err := reference.GetReference(scheme.Scheme, objOrRef)
	if err != nil {
		return nil, err
	}
	stringRefKind := ref.Kind
	var refKind *string
	if len(stringRefKind) > 0 {
		refKind = &stringRefKind
	}
	stringRefUID := string(ref.UID)
	var refUID *string
	if len(stringRefUID) > 0 {
		refUID = &stringRefUID
	}

	e := client.Events(ref.Namespace)
	fieldSelector := e.GetFieldSelector(&ref.Name, &ref.Namespace, refKind, refUID)
	initialOpts := metav1.ListOptions{FieldSelector: fieldSelector.String()}
	eventList := &corev1.EventList{}
	err = FollowContinue(&initialOpts,
		func(options metav1.ListOptions) (runtime.Object, error) {
			newEvents, err := e.List(context.TODO(), options)
			if err != nil {
				return nil, EnhanceListError(err, options, "events")
			}
			eventList.Items = append(eventList.Items, newEvents.Items...)
			return newEvents, nil
		})
	return eventList, err
}

func FollowContinue(initialOpts *metav1.ListOptions,
	listFunc func(metav1.ListOptions) (runtime.Object, error)) error {
	opts := initialOpts
	for {
		list, err := listFunc(*opts)
		if err != nil {
			return err
		}
		nextContinueToken, _ := meta.NewAccessor().Continue(list)
		if len(nextContinueToken) == 0 {
			return nil
		}
		opts.Continue = nextContinueToken
	}
}

func EnhanceListError(err error, opts metav1.ListOptions, subj string) error {
	if apierrors.IsResourceExpired(err) {
		return err
	}
	if apierrors.IsBadRequest(err) || apierrors.IsNotFound(err) {
		if se, ok := err.(*apierrors.StatusError); ok {
			// modify the message without hiding this is an API error
			if len(opts.LabelSelector) == 0 && len(opts.FieldSelector) == 0 {
				se.ErrStatus.Message = fmt.Sprintf("Unable to list %q: %v", subj,
					se.ErrStatus.Message)
			} else {
				se.ErrStatus.Message = fmt.Sprintf(
					"Unable to find %q that match label selector %q, field selector %q: %v", subj,
					opts.LabelSelector,
					opts.FieldSelector, se.ErrStatus.Message)
			}
			return se
		}
		if len(opts.LabelSelector) == 0 && len(opts.FieldSelector) == 0 {
			return fmt.Errorf("Unable to list %q: %v", subj, err)
		}
		return fmt.Errorf("Unable to find %q that match label selector %q, field selector %q: %v",
			subj, opts.LabelSelector, opts.FieldSelector, err)
	}
	return err
}
