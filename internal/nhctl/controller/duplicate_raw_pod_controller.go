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
	"strings"
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

	suffix := r.Identifier[0:5]
	labelsMap, err := r.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	originalPod.Name = strings.Join([]string{originalPod.Name, suffix}, "-")
	originalPod.Labels = labelsMap
	originalPod.Status = corev1.PodStatus{}
	originalPod.ResourceVersion = ""

	devContainer, err := findContainerInPodSpec(originalPod, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		r.genContainersAndVolumes(devContainer, ops.Container, ops.DevImage, ops.StorageClass, true)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range originalPod.Spec.Containers {
			if c.Name == ops.Container {
				originalPod.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		originalPod.Spec.Containers[0] = *devContainer
	}

	// Add volumes to spec
	if originalPod.Spec.Volumes == nil {
		originalPod.Spec.Volumes = make([]corev1.Volume, 0)
	}
	originalPod.Spec.Volumes = append(originalPod.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	originalPod.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(originalPod.Spec.Containers); i++ {
		originalPod.Spec.Containers[i].LivenessProbe = nil
		originalPod.Spec.Containers[i].ReadinessProbe = nil
		originalPod.Spec.Containers[i].StartupProbe = nil
		originalPod.Spec.Containers[i].SecurityContext = nil
	}

	originalPod.Spec.Containers = append(originalPod.Spec.Containers, *sideCarContainer)

	log.Info("Create duplicate dev pod...")
	if _, err = r.Client.CreatePod(originalPod); err != nil {
		return err
	}

	return waitingPodToBeReady(r.GetNocalhostDevContainerPod)
}

func (r *DuplicateRawPodController) RollBack(reset bool) error {
	deploys, err := r.GetPodList()
	if err != nil {
		return err
	}
	if len(deploys) != 1 {
		return errors.New("Duplicated dev pod num is not 1?")
	}
	if err = r.Client.DeletePod(deploys[0].Name, false, 1*time.Second); err != nil {
		return err
	}
	return r.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.DevModeType = ""
		return nil
	})

	//originPod, err := r.Client.GetPod(r.GetName())
	//if err != nil {
	//	return err
	//}
	//podSpec, ok := originPod.Annotations[originalPodDefine]
	//if !ok {
	//	err1 := errors.New(fmt.Sprintf("Annotation %s not found, failed to rollback", originalPodDefine))
	//	if reset {
	//		log.WarnE(err1, "")
	//		return nil
	//	}
	//	return err1
	//}
	//
	//originPod = &corev1.Pod{}
	//
	//if err = json.Unmarshal([]byte(podSpec), originPod); err != nil {
	//	return err
	//}
	//
	//log.Info(" Deleting current revision...")
	//if err = r.Client.DeletePodByName(r.GetName(), 0); err != nil {
	//	return err
	//}
	//
	//log.Info(" Recreating original revision...")
	//if _, err = r.Client.CreatePod(originPod); err != nil {
	//	return err
	//}
	//return nil
}

func (r *DuplicateRawPodController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := r.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	return r.Client.Labels(labelsMap).ListPods()
}
