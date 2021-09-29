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
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"strings"
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

	if s.IsInDevMode() {
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

	p, err := s.GetAppProfile()
	if err != nil {
		return err
	}
	if p.Identifier == "" {
		return errors.New("Identifier can not be nil ")
	}

	suffix := p.Identifier[0:5]
	labelsMap, err := s.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	dep.Name = strings.Join([]string{dep.Name, suffix}, "-")
	dep.Labels = labelsMap
	dep.Status = v1.StatefulSetStatus{}
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	dep.Spec.Template.Labels = labelsMap
	dep.ResourceVersion = ""

	devContainer, err := findContainerInStatefulSetsSpec(dep, ops.Container)
	if err != nil {
		return err
	}

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := s.generateSyncVolumesAndMounts(true)
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := s.genWorkDirAndPVAndMounts(
		ops.Container, ops.StorageClass, true)
	if err != nil {
		return err
	}

	devModeVolumes = append(devModeVolumes, workDirAndPersistVolumes...)
	devModeMounts = append(devModeMounts, workDirAndPersistVolumeMounts...)

	workDir := s.GetWorkDir(ops.Container)
	devImage := s.GetDevImage(ops.Container) // Default : replace the first container
	if devImage == "" {
		return errors.New("Dev image must be specified")
	}

	sideCarContainer := generateSideCarContainer(s.GetDevSidecarImage(ops.Container), workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// set image pull policy
	sideCarContainer.ImagePullPolicy = _const.DefaultSidecarImagePullPolicy
	devContainer.ImagePullPolicy = _const.DefaultSidecarImagePullPolicy

	// add env
	devEnv := s.GetDevContainerEnv(ops.Container)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := s.genResourceReq(ops.Container)
	if requirements != nil {
		devContainer.Resources = *requirements
	}
	r := &profile.ResourceQuota{
		Limits:   &profile.QuotaList{Memory: "1Gi", Cpu: "1"},
		Requests: &profile.QuotaList{Memory: "50Mi", Cpu: "100m"},
	}
	rq, _ := convertResourceQuota(r)
	sideCarContainer.Resources = *rq

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

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	log.Info("Create duplicate StateFulSets...")

	_, err = s.Client.CreateStatefulSetAndWait(dep)
	if err != nil {
		return err
	}
	return waitingPodToBeReady(s.GetNocalhostDevContainerPod)
}

// Container Get specify container
// If containerName not specified:
// 	 if there is only one container defined in spec, return it
//	 if there are more than one container defined in spec, return err
func findContainerInStatefulSetsSpec(ss *v1.StatefulSet, containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container
	if containerName != "" {
		for index, c := range ss.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				return &ss.Spec.Template.Spec.Containers[index], nil
			}
		}
		if devContainer == nil {
			return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
		}
	} else {
		if len(ss.Spec.Template.Spec.Containers) > 1 {
			return nil, errors.New(
				fmt.Sprintf(
					"There are more than one container defined, " +
						"please specify one to start developing",
				),
			)
		}
		if len(ss.Spec.Template.Spec.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &ss.Spec.Template.Spec.Containers[0]
	}
	return devContainer, nil
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
		return errors.New("StatefulSets num is not 1?")
	}
	if err = s.Client.DeleteStatefulSet(ss[0].Name); err != nil {
		return err
	}
	return s.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.LocalDevMode = ""
		svcProfileV2.DuplicateDevMode = false
		return nil
	})
}

func (s *DuplicateStatefulSetController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := s.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	s.Client.Labels(labelsMap)
	return s.Client.ListPods()
}
