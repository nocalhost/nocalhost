/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
	"time"
)

type DuplicateRawPodController struct {
	*Controller
}

func (r *DuplicateRawPodController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	r.Client.Context(ctx)
	originalPod, err := r.Client.GetPod(r.Name)
	if err != nil {
		return err
	}

	// Check if pod managed by controller
	if len(originalPod.OwnerReferences) > 0 {
		return errors.New(fmt.Sprintf("Pod %s is manged by a controller, can not enter DevMode", r.Name))
	}

	if r.IsInReplaceDevMode() {
		if len(originalPod.Annotations) > 0 {
			podSpec, ok := originalPod.Annotations[_const.OriginWorkloadDefinition]
			if !ok {
				return errors.New(
					fmt.Sprintf(
						"Annotation %s not found, failed to rollback", _const.OriginWorkloadDefinition,
					),
				)
			}
			if err = json.Unmarshal([]byte(podSpec), originalPod); err != nil {
				return errors.Wrap(err, "")
			}
		} else {
			return errors.New(
				fmt.Sprintf(
					"Annotation %s not found, failed to rollback", _const.OriginWorkloadDefinition,
				),
			)
		}
	} else {
		if len(originalPod.Annotations) > 0 {
			podSpec, ok := originalPod.Annotations[_const.OriginWorkloadDefinition]
			var oPodSpec = corev1.Pod{}
			if ok {
				if err = json.Unmarshal([]byte(podSpec), &oPodSpec); err == nil {
					originalPod = &oPodSpec
				}
			}
		}
	}

	labelsMap := r.getDuplicateLabelsMap()

	originalPod.Name = r.getDuplicateResourceName()
	originalPod.Labels = labelsMap
	originalPod.Status = corev1.PodStatus{}
	originalPod.ResourceVersion = ""
	originalPod.Annotations = r.getDevContainerAnnotations(ops.Container, originalPod.Annotations)

	devContainer, sideCarContainer, devModeVolumes, err :=
		r.genContainersAndVolumes(&originalPod.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&originalPod.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	log.Info("Create duplicate dev pod...")
	if _, err = r.Client.CreatePod(originalPod); err != nil {
		return err
	}

	r.patchAfterDevContainerReplaced(ops.Container, originalPod.Kind, originalPod.Name)

	r.waitDevPodToBeReady()
	return nil
}

func (r *DuplicateRawPodController) RollBack(reset bool) error {
	deploys, err := r.GetPodList()
	if err != nil {
		return err
	}

	if len(deploys) != 1 {
		if !reset {
			return errors.New(fmt.Sprintf("Duplicate pod num is %d (not 1)?", len(deploys)))
		} else if len(deploys) == 0 {
			log.Warnf("Duplicate pod num is %d (not 1)?", len(deploys))
			return nil
		}
	}

	return r.Client.DeletePod(deploys[0].Name, false, 1*time.Second)
}
