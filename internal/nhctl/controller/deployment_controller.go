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
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/pod_controller"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

type DeploymentController struct {
	*Controller
}

func (d *DeploymentController) Name() string {
	return d.Controller.Name
}

// ReplaceImage In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (d *DeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	var err error
	d.Client.Context(ctx)

	if err = d.markReplicaSetRevision(); err != nil {
		return err
	}

	if err = d.ScaleReplicasToOne(ctx); err != nil {
		return err
	}

	devContainer, err := d.Container(ops.Container)
	if err != nil {
		return err
	}

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := d.generateSyncVolumesAndMounts()
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := d.genWorkDirAndPVAndMounts(
		ops.Container, ops.StorageClass)
	if err != nil {
		return err
	}

	devModeVolumes = append(devModeVolumes, workDirAndPersistVolumes...)
	devModeMounts = append(devModeMounts, workDirAndPersistVolumeMounts...)

	workDir := d.GetWorkDir(ops.Container)
	devImage := d.GetDevImage(ops.Container) // Default : replace the first container

	sideCarContainer := generateSideCarContainer(workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// add env
	devEnv := d.GetDevContainerEnv(ops.Container)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := d.genResourceReq()
	if requirements != nil {
		devContainer.Resources = *requirements
		sideCarContainer.Resources = *requirements
	}

	for i := 0; i < 10; i++ {
		// Get latest deployment
		dep, err := d.Client.GetDeployment(d.Name())
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

		// PriorityClass
		priorityClass := ops.PriorityClass
		if priorityClass == "" {
			svcProfile, _ := d.GetProfile()
			priorityClass = svcProfile.PriorityClass
		}
		if priorityClass != "" {
			log.Infof("Using priorityClass: %s...", priorityClass)
			dep.Spec.Template.Spec.PriorityClassName = priorityClass
		}

		log.Info("Updating development container...")
		_, err = d.Client.UpdateDeployment(dep, true)
		if err != nil {
			if strings.Contains(err.Error(), "no PriorityClass") {
				log.Warnf("PriorityClass %s not found, disable it...", priorityClass)
				dep, err = d.Client.GetDeployment(d.Name())
				if err != nil {
					return err
				}
				dep.Spec.Template.Spec.PriorityClassName = ""
				_, err = d.Client.UpdateDeployment(dep, true)
				if err != nil {
					if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
						log.Warn("Deployment has been modified, retrying...")
						continue
					}
					return err
				}
				break
			} else if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
				log.Warn("Deployment has been modified, retrying...")
				continue
			}
			return err
		}
	}
	return d.waitingPodToBeReady()

}

func (d *DeploymentController) ScaleReplicasToOne(ctx context.Context) error {

	deploymentsClient := d.Client.GetDeploymentClient()
	scale, err := deploymentsClient.GetScale(ctx, d.Name(), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 {
		log.Infof("Deployment %s replicas is %d now, scaling it to 1", d.Name(), scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(ctx, d.Name(), scale, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "")
		} else {
			//for i := 0; i < 60; i++ {
			//	time.Sleep(time.Second * 1)
			//	scale, err = deploymentsClient.GetScale(ctx, d.Name(), metav1.GetOptions{})
			//	if scale.Spec.Replicas > 1 {
			//		log.Debugf("Waiting replicas scaling to 1")
			//	} else {
			//		break
			//	}
			//}
			log.Info("Replicas has been set to 1")
		}
	} else {
		log.Infof("Deployment %s replicas is already 1, no need to scale", d.Name())
	}
	return nil
}

func (d *DeploymentController) Container(containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	dep, err := d.Client.GetDeployment(d.Name())
	if err != nil {
		return nil, err
	}
	if containerName != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				return &dep.Spec.Template.Spec.Containers[index], nil
			}
		}
		if devContainer == nil {
			return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
		}
	} else {
		if len(dep.Spec.Template.Spec.Containers) > 1 {
			return nil, errors.New(fmt.Sprintf("There are more than one container defined," +
				"please specify one to start developing"))
		}
		if len(dep.Spec.Template.Spec.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &dep.Spec.Template.Spec.Containers[0]
	}
	return devContainer, nil
}

func (d *DeploymentController) RollBack(reset bool) error {
	clientUtils := d.Client

	dep, err := clientUtils.GetDeployment(d.Name())
	if err != nil {
		return err
	}

	rss, err := clientUtils.GetSortedReplicaSetsByDeployment(d.Name())
	if err != nil {
		log.WarnE(err, "Failed to get rs list")
		return err
	}

	// Find previous replicaSet
	if len(rss) < 2 {
		log.Warn("No history to roll back")
		return nil
	}

	var r *v1.ReplicaSet
	var originalPodReplicas *int32
	for _, rs := range rss {
		if rs.Annotations == nil {
			continue
		}
		// Mark the original revision
		if rs.Annotations[nocalhost.DevImageRevisionAnnotationKey] == nocalhost.DevImageRevisionAnnotationValue {
			r = rs
			if rs.Annotations[nocalhost.DevImageOriginalPodReplicasAnnotationKey] != "" {
				podReplicas, _ := strconv.Atoi(rs.Annotations[nocalhost.DevImageOriginalPodReplicasAnnotationKey])
				podReplicas32 := int32(podReplicas)
				originalPodReplicas = &podReplicas32
			}
		}
	}
	if r == nil {
		if !reset {
			return errors.New("Failed to find the proper revision to rollback")
		} else {
			r = rss[0]
		}
	}

	dep.Spec.Template = r.Spec.Template
	if originalPodReplicas != nil {
		dep.Spec.Replicas = originalPodReplicas
	}

	log.Info(" Deleting current revision...")
	err = clientUtils.DeleteDeployment(dep.Name, false)
	if err != nil {
		return err
	}

	log.Info(" Recreating original revision...")
	dep.ResourceVersion = ""
	if len(dep.Annotations) == 0 {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations["nocalhost-dep-ignore"] = "true"

	// Add labels and annotations
	if dep.Labels == nil {
		dep.Labels = make(map[string]string, 0)
	}
	dep.Labels[nocalhost.AppManagedByLabel] = nocalhost.AppManagedByNocalhost

	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations[nocalhost.NocalhostApplicationName] = d.AppName
	dep.Annotations[nocalhost.NocalhostApplicationNamespace] = d.NameSpace

	_, err = clientUtils.CreateDeployment(dep)
	if err != nil {
		if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
			log.Warn("[Warning] Nocalhost-dep needs to update")
		}
		return err
	}

	return nil
}

func (d *DeploymentController) GetDefaultPodNameWait(ctx context.Context) (string, error) {
	return getDefaultPodName(ctx, d)
}

func getDefaultPodName(ctx context.Context, p pod_controller.PodController) (string, error) {
	var (
		podList []corev1.Pod
		err     error
	)
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", p.Name()))
		default:
			podList, err = p.GetPodList()
		}
		if err != nil || len(podList) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", p.Name())
			time.Sleep(time.Second)
		} else {
			return podList[0].Name, nil
		}
	}
}

func (d *DeploymentController) GetPodList() ([]corev1.Pod, error) {
	return d.Client.ListLatestRevisionPodsByDeployment(d.Name())
}
