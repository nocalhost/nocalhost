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
	"nocalhost/pkg/nhctl/log"
	"time"
)

const originalPodDefine = "nocalhost.dev.origin.pod.define"

// RawPodController represents a pod not managed by any controller
type RawPodController struct {
	*Controller
}

func (r *RawPodController) GetNocalhostDevContainerPod() (string, error) {
	pod, err := r.Client.GetPod(r.Name())
	if err != nil {
		return "", err
	}
	checkPodsList := []corev1.Pod{*pod}
	return findDevPod(checkPodsList)
}

func (r *RawPodController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	r.Client.Context(ctx)
	originalPod, err := r.Client.GetPod(r.Name())
	if err != nil {
		return err
	}

	// Check if pod managed by controller
	if len(originalPod.OwnerReferences) > 0 {
		return errors.New(fmt.Sprintf("Pod %s is manged by a controller, can not enter DevMode", r.Name()))
	}

	originalPod.Status = corev1.PodStatus{}
	originalPod.ResourceVersion = ""

	bys, err := json.Marshal(originalPod)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if originalPod.Annotations == nil {
		originalPod.Annotations = make(map[string]string, 0)
	}
	originalPod.Annotations[originalPodDefine] = string(bys)

	devContainer, err := findContainerInPodSpec(originalPod, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		r.genContainersAndVolumes(devContainer, ops.Container, ops.DevImage, ops.StorageClass)
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

	log.Info("Delete original pod...")
	if err = r.Client.DeletePodByName(r.Name(), 0); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	//log.Info("Waiting pod to be deleted")
	//if err = k8sutils.WaitPodDeleted(r.Client.ClientSet, r.NameSpace, r.Name(), 10*time.Minute); err != nil {
	//	log.Info("Recovering pod...")
	//	if _, err := r.Client.CreatePod(originalPod); err != nil {
	//		return err
	//	}
	//	return err
	//}

	log.Info("Create dev pod...")
	if _, err = r.Client.CreatePod(originalPod); err != nil {
		return err
	}

	return waitingPodToBeReady(r.GetNocalhostDevContainerPod)
}

func (r *RawPodController) Name() string {
	return r.Controller.Name
}

func (r *RawPodController) RollBack(reset bool) error {
	originPod, err := r.Client.GetPod(r.Name())
	if err != nil {
		return err
	}
	podSpec, ok := originPod.Annotations[originalPodDefine]
	if !ok {
		err1 := errors.New(fmt.Sprintf("Annotation %s not found, failed to rollback", originalPodDefine))
		if reset {
			log.WarnE(err1, "")
			return nil
		}
		return err1
	}

	originPod = &corev1.Pod{}

	if err = json.Unmarshal([]byte(podSpec), originPod); err != nil {
		return err
	}

	log.Info(" Deleting current revision...")
	if err = r.Client.DeletePodByName(r.Name(), 0); err != nil {
		return err
	}

	log.Info(" Recreating original revision...")
	if _, err = r.Client.CreatePod(originPod); err != nil {
		return err
	}
	return nil
}

func (r *RawPodController) GetPodList() ([]corev1.Pod, error) {
	pod, err := r.Client.GetPod(r.Name())
	if err != nil {
		return nil, err
	}
	return []corev1.Pod{*pod}, nil
}

func findContainerInPodSpec(pod *corev1.Pod, containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	if containerName != "" {
		for index, c := range pod.Spec.Containers {
			if c.Name == containerName {
				return &pod.Spec.Containers[index], nil
			}
		}
		return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
	} else {
		if len(pod.Spec.Containers) > 1 {
			return nil, errors.New(
				fmt.Sprintf(
					"There are more than one container defined," +
						"please specify one to start developing",
				),
			)
		}
		if len(pod.Spec.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &pod.Spec.Containers[0]
	}
	return devContainer, nil
}
