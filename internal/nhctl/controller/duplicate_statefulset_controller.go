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
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
	"time"
)

type DuplicateStatefulSetController struct {
	*Controller
}

func (s *DuplicateStatefulSetController) GetNocalhostDevContainerPod() (string, error) {
	pods, err := s.GetPodList()
	if err != nil {
		return "", err
	}

	return findDevPod(pods)
}

func (s *DuplicateStatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	var err error
	s.Client.Context(ctx)

	dep, err := s.Client.GetStatefulSet(s.Name)
	if err != nil {
		return err
	}

	if s.IsInReplaceDevMode() {
		osj, ok := dep.Annotations[OriginSpecJson]
		if ok {
			log.Info("Annotation nocalhost.origin.spec.json found, use it")
			dep.Spec = v1.StatefulSetSpec{}
			if err = json.Unmarshal([]byte(osj), &dep.Spec); err != nil {
				return errors.Wrap(err, "")
			}
		} else {
			return errors.New("Annotation nocalhost.origin.spec.json not found?")
		}
	}

	var rs int32 = 1
	dep.Spec.Replicas = &rs

	labelsMap, err := s.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	dep.Name = s.getDuplicateResourceName()
	dep.Labels = labelsMap
	dep.Status = v1.StatefulSetStatus{}
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	dep.Spec.Template.Labels = labelsMap
	dep.ResourceVersion = ""

	devContainer, sideCarContainer, devModeVolumes, err :=
		s.genContainersAndVolumes(&dep.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				dep.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		dep.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if dep.Spec.Template.Spec.Volumes == nil {
		log.Debugf("Service %s has no volume", dep.Name)
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
		dep.Spec.Template.Spec.Containers[i].SecurityContext = nil
	}

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *sideCarContainer)

	log.Info("Create duplicate StateFulSets...")

	_, err = s.Client.CreateStatefulSetAndWait(dep)
	if err != nil {
		return err
	}

	s.patchAfterDevContainerReplaced(ops.Container, dep.Kind, dep.Name)
	<-time.Tick(time.Second)

	return waitingPodToBeReady(s.GetNocalhostDevContainerPod)
}

func (s *DuplicateStatefulSetController) RollBack(reset bool) error {
	lmap, err := s.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	s.Client.Labels(lmap)
	ss, err := s.Client.ListStatefulSets()
	if err != nil {
		return err
	}

	if len(ss) != 1 {
		if !reset {
			return errors.New(fmt.Sprintf("Duplicate StatefulSet num is %d (not 1)?", len(ss)))
		} else if len(ss) == 0 {
			log.Warnf("Duplicate StatefulSet num is %d (not 1)?", len(ss))
			return nil
		}
	}

	return s.Client.DeleteStatefulSet(ss[0].Name)
}

func (s *DuplicateStatefulSetController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := s.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	s.Client.Labels(labelsMap)
	return s.Client.ListPods()
}
