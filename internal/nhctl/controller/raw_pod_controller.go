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
	pod, err := r.Client.GetPod(r.GetName())
	if err != nil {
		return "", err
	}
	checkPodsList := []corev1.Pod{*pod}
	return findDevPodName(checkPodsList...)
}

func (r *RawPodController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	r.Client.Context(ctx)
	originalPod, err := r.Client.GetPod(r.GetName())
	if err != nil {
		return err
	}

	// Check if pod managed by controller
	if len(originalPod.OwnerReferences) > 0 {
		return errors.New(fmt.Sprintf("Pod %s is manged by a controller, can not enter DevMode", r.GetName()))
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

	devContainer, sideCarContainer, devModeVolumes, err :=
		r.genContainersAndVolumes(&originalPod.Spec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&originalPod.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	log.Info("Delete original pod...")
	if err = r.Client.DeletePodByName(r.GetName(), 0); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	log.Info("Create dev pod...")
	if _, err = r.Client.CreatePod(originalPod); err != nil {
		return err
	}

	r.patchAfterDevContainerReplaced(ops.Container, originalPod.Kind, originalPod.Name)

	return waitingPodToBeReady(r.GetNocalhostDevContainerPod)
}

func (r *RawPodController) RollBack(reset bool) error {
	originPod, err := r.Client.GetPod(r.GetName())
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
	if err = r.Client.DeletePodByName(r.GetName(), 0); err != nil {
		return err
	}

	log.Info(" Recreating original revision...")
	if _, err = r.Client.CreatePod(originPod); err != nil {
		return err
	}
	return nil
}

func (r *RawPodController) GetPodList() ([]corev1.Pod, error) {
	pod, err := r.Client.GetPod(r.GetName())
	if err != nil {
		return nil, err
	}
	return []corev1.Pod{*pod}, nil
}

func findDevContainerInPodSpec(pod *corev1.PodSpec, containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	if containerName != "" {
		for index, c := range pod.Containers {
			if c.Name == containerName {
				return &pod.Containers[index], nil
			}
		}
		return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
	} else {
		if len(pod.Containers) > 1 {
			return nil, errors.New(
				fmt.Sprintf(
					"There are more than one container defined," +
						"please specify one to start developing",
				),
			)
		}
		if len(pod.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &pod.Containers[0]
	}
	return devContainer, nil
}
