/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package remote

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/vpn/util"
	"strconv"
)

var CancelFunctions = make([]context.CancelFunc, 3)

// UpdateRefCount vendor/k8s.io/kubectl/pkg/polymorphichelpers/rollback.go:99
func UpdateRefCount(clientset *kubernetes.Clientset, namespace, name string, increment int) {
	if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !errors.IsNotFound(err)
	}, func() error {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			log.Errorf("update ref-count failed, increment: %d, error: %v", increment, err)
			return err
		}
		curCount := 0
		if ref := pod.GetAnnotations()["ref-count"]; len(ref) == 0 {
			return fmt.Errorf("can't found ref-count from pod annotation, this should not happend")
		} else if curCount, err = strconv.Atoi(ref); err != nil {
			return err
		}
		patch, _ := json.Marshal([]interface{}{
			map[string]interface{}{
				"op":    "replace",
				"path":  "/metadata/annotations/ref-count",
				"value": strconv.Itoa(curCount + increment),
			},
		})
		_, err = clientset.CoreV1().Pods(namespace).
			Patch(context.TODO(), util.TrafficManager, types.JSONPatchType, patch, v1.PatchOptions{})
		return err
	}); err != nil {
		log.Errorf("update ref count error, error: %v", err)
	} else {
		log.Infof("update ref count successfully, increment: %v", increment)
	}
}

func CleanUpTrafficManagerIfRefCountIsZero(clientset *kubernetes.Clientset, namespace string) {
	UpdateRefCount(clientset, namespace, util.TrafficManager, -1)
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		log.Error(err)
		return
	}
	refCount, err := strconv.Atoi(pod.GetAnnotations()["ref-count"])
	if err != nil {
		log.Error(err)
		return
	}
	// if refcount is less than zero or equals to zero, means no body will using this dns pod, so clean it
	if refCount <= 0 {
		zero := int64(0)
		log.Info("refCount is zero, prepare to clean up resource")
		_ = clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), util.TrafficManager, v1.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		_ = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), util.TrafficManager, v1.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
	}
}
