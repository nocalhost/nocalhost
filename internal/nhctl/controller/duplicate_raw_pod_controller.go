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
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"time"
)

type DuplicateRawPodController struct {
	*Controller
}

func (r *DuplicateRawPodController) GetNocalhostDevContainerPod() (string, error) {
	pods, err := r.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPod(pods)
}

func (r *DuplicateRawPodController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	r.Client.Context(ctx)
	originalPod, err := r.Client.GetPod(r.GetName())
	if err != nil {
		return err
	}

	// Check if pod managed by controller
	if len(originalPod.OwnerReferences) > 0 {
		return errors.New(fmt.Sprintf("Pod %s is manged by a controller, can not enter DevMode", r.GetName()))
	}

	if r.IsInReplaceDevMode() {
		if len(originalPod.Annotations) > 0 {
			podSpec, ok := originalPod.Annotations[originalPodDefine]
			if !ok {
				return errors.New(fmt.Sprintf("Annotation %s not found, failed to rollback", originalPodDefine))
			}
			if err = json.Unmarshal([]byte(podSpec), originalPod); err != nil {
				return errors.Wrap(err, "")
			}
		} else {
			return errors.New(fmt.Sprintf("Annotation %s not found, failed to rollback", originalPodDefine))
		}
	} else {
		if len(originalPod.Annotations) > 0 {
			podSpec, ok := originalPod.Annotations[originalPodDefine]
			var oPodSpec = corev1.Pod{}
			if ok {
				if err = json.Unmarshal([]byte(podSpec), &oPodSpec); err == nil {
					originalPod = &oPodSpec
				}
			}
		}
	}

	labelsMap, err := r.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	originalPod.Name = r.getDuplicateResourceName()
	originalPod.Labels = labelsMap
	originalPod.Status = corev1.PodStatus{}
	originalPod.ResourceVersion = ""

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

	for _, patch := range r.config.GetContainerDevConfigOrDefault(ops.Container).Patches {
		log.Infof("Patching %s", patch.Patch)
		if err = r.Client.Patch(r.Type.String(), originalPod.Name, patch.Patch, patch.Type); err != nil {
			log.WarnE(err, "")
		}
	}
	<-time.Tick(time.Second)

	return waitingPodToBeReady(r.GetNocalhostDevContainerPod)
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
			_ = r.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
				svcProfileV2.DevModeType = ""
				return nil
			})
			return nil
		}
	}

	if err = r.Client.DeletePod(deploys[0].Name, false, 1*time.Second); err != nil {
		return err
	}
	return r.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.DevModeType = ""
		return nil
	})
}

func (r *DuplicateRawPodController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := r.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	return r.Client.Labels(labelsMap).ListPods()
}
