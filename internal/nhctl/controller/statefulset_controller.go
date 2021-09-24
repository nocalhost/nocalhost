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
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"time"
)

const (
	OriginSpecJson = "nocalhost.origin.spec.json"
)

type StatefulSetController struct {
	*Controller
}

func (s *StatefulSetController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := s.Client.ListPodsByStatefulSet(s.GetName())
	if err != nil {
		return "", err
	}

	return findDevPod(checkPodsList.Items)
}

//func (s *StatefulSetController) Name() string {
//	return s.Controller.Name
//}

func (s *StatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	var err error
	s.Client.Context(ctx)

	dep, err := s.Client.GetStatefulSet(s.Name)
	if err != nil {
		return err
	}

	originalSpecJson, err := json.Marshal(&dep.Spec)
	if err != nil {
		return errors.Wrap(err, "")
	}

	//if err = s.ScaleReplicasToOne(ctx); err != nil {
	//	return err
	//}
	s.Client.Context(ctx)
	if err = s.Client.ScaleStatefulSetReplicasToOne(s.GetName()); err != nil {
		return err
	}

	devContainer, err := s.Container(ops.Container)
	if err != nil {
		return err
	}

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := s.generateSyncVolumesAndMounts()
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := s.genWorkDirAndPVAndMounts(
		ops.Container, ops.StorageClass,
	)
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

	needToRemovePriorityClass := false
	for i := 0; i < 10; i++ {
		events, err := s.Client.ListEventsByStatefulSet(s.GetName())
		utils.Should(err)
		_ = s.Client.DeleteEvents(events, true)

		// Get the latest stateful set
		dep, err = s.Client.GetStatefulSet(s.GetName())
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

		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

		if len(dep.Annotations) == 0 {
			dep.Annotations = make(map[string]string, 0)
		}
		dep.Annotations[OriginSpecJson] = string(originalSpecJson)

		log.Info("Updating development container...")

		_, err = s.Client.UpdateStatefulSet(dep, true)
		if err != nil {
			if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
				log.Warn("StatefulSet has been modified, retrying...")
				continue
			}
			return err
		} else {
			// Check if priorityClass exists
		outer:
			for i := 0; i < 20; i++ {
				time.Sleep(1 * time.Second)
				events, err = s.Client.ListEventsByStatefulSet(s.GetName())
				for _, event := range events {
					if strings.Contains(event.Message, "no PriorityClass") {
						log.Warn("PriorityClass not found, disable it...")
						needToRemovePriorityClass = true
						break outer
					} else if event.Reason == "SuccessfulCreate" {
						log.Infof("Pod SuccessfulCreate")
						break outer
					}
				}
			}

			if needToRemovePriorityClass {
				dep, err = s.Client.GetStatefulSet(s.GetName())
				if err != nil {
					return err
				}
				dep.Spec.Template.Spec.PriorityClassName = ""
				log.Info("Removing priorityClass")
				_, err = s.Client.UpdateStatefulSet(dep, true)
				if err != nil {
					if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
						log.Warn("StatefulSet has been modified, retrying...")
						continue
					}
					return err
				}
				break
			}
		}
		break
	}
	return waitingPodToBeReady(s.GetNocalhostDevContainerPod)
}

// Container Get specify container
// If containerName not specified:
// 	 if there is only one container defined in spec, return it
//	 if there are more than one container defined in spec, return err
func (s *StatefulSetController) Container(containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	ss, err := s.Client.GetStatefulSet(s.GetName())
	if err != nil {
		return nil, err
	}
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

func (s *StatefulSetController) RollBack(reset bool) error {
	clientUtils := s.Client

	dep, err := clientUtils.GetStatefulSet(s.GetName())
	if err != nil {
		return err
	}

	osj, ok := dep.Annotations[OriginSpecJson]
	if !ok {
		return errors.New("No spec json found to rollback")
	}

	log.Info(" Deleting current revision...")
	if err = clientUtils.DeleteStatefulSet(dep.Name); err != nil {
		return err
	}

	dep.Spec = v1.StatefulSetSpec{}
	if err = json.Unmarshal([]byte(osj), &dep.Spec); err != nil {
		return errors.Wrap(err, "")
	}

	log.Info(" Recreating original revision...")
	dep.ResourceVersion = ""
	if len(dep.Annotations) == 0 {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations["nocalhost-dep-ignore"] = "true"
	dep.Annotations[OriginSpecJson] = osj

	// Add labels and annotations
	if dep.Labels == nil {
		dep.Labels = make(map[string]string, 0)
	}
	dep.Labels[_const.AppManagedByLabel] = _const.AppManagedByNocalhost

	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations[_const.NocalhostApplicationName] = s.AppName
	dep.Annotations[_const.NocalhostApplicationNamespace] = s.NameSpace

	_, err = clientUtils.CreateStatefulSet(dep)
	if err != nil {
		if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
			log.Warn("[Warning] Nocalhost-dep needs to update")
		}
		return err
	}
	return nil
}

func (s *StatefulSetController) GetPodList() ([]corev1.Pod, error) {
	list, err := s.Client.ListPodsByStatefulSet(s.GetName())
	if err != nil {
		return nil, err
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("no pod found")
	}
	return list.Items, nil
}
