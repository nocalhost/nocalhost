/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
	"nocalhost/internal/nhctl/nocalhost"
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

func (s *StatefulSetController) Name() string {
	return s.Controller.Name
}

func (s *StatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	var err error
	s.Client.Context(ctx)

	dep, err := s.Client.GetStatefulSet(s.Name())
	if err != nil {
		return err
	}

	originalSpecJson, err := json.Marshal(&dep.Spec)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err = s.ScaleReplicasToOne(ctx); err != nil {
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
		ops.Container, ops.StorageClass)
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

	sideCarContainer := generateSideCarContainer(workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// set image pull policy
	sideCarContainer.ImagePullPolicy = nocalhost.DefaultSidecarImagePullPolicy
	devContainer.ImagePullPolicy = nocalhost.DefaultSidecarImagePullPolicy

	// add env
	devEnv := s.GetDevContainerEnv(ops.Container)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := s.genResourceReq()
	if requirements != nil {
		devContainer.Resources = *requirements
		sideCarContainer.Resources = *requirements
	}

	needToRemovePriorityClass := false
	for i := 0; i < 10; i++ {
		events, err := s.Client.ListEventsByStatefulSet(s.Name())
		utils.Should(err)
		_ = s.Client.DeleteEvents(events, true)

		// Get the latest stateful set
		dep, err = s.Client.GetStatefulSet(s.Name())
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
		}

		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

		// todo PriorityClass
		//priorityClass := ops.PriorityClass
		//if priorityClass == "" {
		//	svcProfile, _ := s.GetProfile()
		//	priorityClass = svcProfile.PriorityClass
		//}
		//if priorityClass != "" && !needToRemovePriorityClass {
		//	log.Infof("Using priorityClass: %s...", priorityClass)
		//	dep.Spec.Template.Spec.PriorityClassName = priorityClass
		//}

		if _, ok := dep.Annotations[OriginSpecJson]; !ok {
			dep.Annotations[OriginSpecJson] = string(originalSpecJson)
		}

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
				events, err = s.Client.ListEventsByStatefulSet(s.Name())
				//log.Infof("Find %d events", len(events))
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
				dep, err = s.Client.GetStatefulSet(s.Name())
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
	return s.waitingPodToBeReady()
}

func (s *StatefulSetController) ScaleReplicasToOne(ctx context.Context) error {
	scale, err := s.Client.GetStatefulSetClient().GetScale(ctx, s.Name(), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 {
		scale.Spec.Replicas = 1
		_, err = s.Client.GetStatefulSetClient().UpdateScale(ctx, s.Name(), scale, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Info("Waiting replicas scale to 1, it may take several minutes...")
		for i := 0; i < 300; i++ {
			time.Sleep(1 * time.Second)
			ss, err := s.Client.GetStatefulSet(s.Name())
			if err != nil {
				return errors.Wrap(err, "")
			}
			if ss.Status.ReadyReplicas == 1 && ss.Status.Replicas == 1 {
				log.Info("Replicas has been scaled to 1")
				return nil
			}
		}
		return errors.New("Waiting replicas scaling to 1 timeout")
	} else {
		log.Info("Replicas has already been scaled to 1")
	}
	return nil
}

// Container Get specify container
// If containerName not specified:
// 	 if there is only one container defined in spec, return it
//	 if there are more than one container defined in spec, return err
func (s *StatefulSetController) Container(containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	ss, err := s.Client.GetStatefulSet(s.Name())
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
			return nil, errors.New(fmt.Sprintf("There are more than one container defined," +
				"please specify one to start developing"))
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

	dep, err := clientUtils.GetStatefulSet(s.Name())
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
	dep.Labels[nocalhost.AppManagedByLabel] = nocalhost.AppManagedByNocalhost

	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations[nocalhost.NocalhostApplicationName] = s.AppName
	dep.Annotations[nocalhost.NocalhostApplicationNamespace] = s.NameSpace

	_, err = clientUtils.CreateStatefulSet(dep)
	if err != nil {
		if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
			log.Warn("[Warning] Nocalhost-dep needs to update")
		}
		return err
	}
	return nil
}

func (s *StatefulSetController) GetDefaultPodNameWait(ctx context.Context) (string, error) {
	return getDefaultPodName(ctx, s)
}

func (s *StatefulSetController) GetPodList() ([]corev1.Pod, error) {
	list, err := s.Client.ListPodsByStatefulSet(s.Name())
	if err != nil {
		return nil, err
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("no pod found")
	}
	return list.Items, nil
}
