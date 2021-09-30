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
	"nocalhost/internal/nhctl/pod_controller"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

type DeploymentController struct {
	*Controller
}

func (d *DeploymentController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := d.Client.ListPodsByDeployment(d.Name())
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList.Items)
}

func (d *DeploymentController) Name() string {
	return d.Controller.Name
}

// ReplaceImage In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (d *DeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	var err error
	d.Client.Context(ctx)

	dep, err := d.Client.GetDeployment(d.Name())
	if err != nil {
		return err
	}

	originalSpecJson, err := json.Marshal(&dep.Spec)
	if err != nil {
		return errors.Wrap(err, "")
	}

	d.Client.Context(ctx)
	if err = d.Client.ScaleDeploymentReplicasToOne(d.Name()); err != nil {
		return err
	}

	devContainer, err := findContainerInDeployment(d.Name(), ops.Container, d.Client)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(devContainer, ops.Container, ops.DevImage, ops.StorageClass)
	if err != nil {
		return err
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
			dep.Spec.Template.Spec.Containers[i].SecurityContext = nil
		}

		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *sideCarContainer)

		// PriorityClass
		priorityClass := ops.PriorityClass
		if priorityClass == "" {
			svcProfile, _ := d.GetConfig()
			priorityClass = svcProfile.PriorityClass
		}
		if priorityClass != "" {
			log.Infof("Using priorityClass: %s...", priorityClass)
			dep.Spec.Template.Spec.PriorityClassName = priorityClass
		}

		dep.Annotations[OriginSpecJson] = string(originalSpecJson)

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
		break
	}
	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)

}

func findContainerInDeployment(deployName, containerName string, client *clientgoutils.ClientGoUtils) (*corev1.Container, error) {
	dep, err := client.GetDeployment(deployName)
	if err != nil {
		return nil, err
	}
	return findContainerInDeploySpec(dep, containerName)
}

func findContainerInDeploySpec(dep *v1.Deployment, containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	if containerName != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				return &dep.Spec.Template.Spec.Containers[index], nil
			}
		}
		return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
	} else {
		if len(dep.Spec.Template.Spec.Containers) > 1 {
			return nil, errors.New(
				fmt.Sprintf(
					"There are more than one container defined, " +
						"please specify one to start developing",
				),
			)
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

	osj, ok := dep.Annotations[OriginSpecJson]
	if ok {
		log.Info("Annotation nocalhost.origin.spec.json found, use it")
		dep.Spec = v1.DeploymentSpec{}
		if err = json.Unmarshal([]byte(osj), &dep.Spec); err != nil {
			return errors.Wrap(err, "")
		}

		if len(dep.Annotations) == 0 {
			dep.Annotations = make(map[string]string, 0)
		}
		dep.Annotations[OriginSpecJson] = osj
	} else {
		log.Info("Annotation nocalhost.origin.spec.json not found, try to find it from rs")
		rss, _ := clientUtils.GetSortedReplicaSetsByDeployment(d.Name())
		if len(rss) >= 1 {
			var r *v1.ReplicaSet
			var originalPodReplicas *int32
			for _, rs := range rss {
				if rs.Annotations == nil {
					continue
				}
				// Mark the original revision
				if rs.Annotations[_const.DevImageRevisionAnnotationKey] == _const.DevImageRevisionAnnotationValue {
					r = rs
					if rs.Annotations[_const.DevImageOriginalPodReplicasAnnotationKey] != "" {
						podReplicas, _ := strconv.Atoi(rs.Annotations[_const.DevImageOriginalPodReplicasAnnotationKey])
						podReplicas32 := int32(podReplicas)
						originalPodReplicas = &podReplicas32
					}
				}
			}

			if r == nil && reset {
				r = rss[0]
			}

			if r != nil {
				dep.Spec.Template = r.Spec.Template
				if originalPodReplicas != nil {
					dep.Spec.Replicas = originalPodReplicas
				}
			} else {
				return errors.New("Failed to find revision to rollout")
			}
		} else {
			return errors.New("Failed to find revision to rollout(no rs found)")
		}
	}

	log.Info(" Deleting current revision...")
	if err = clientUtils.DeleteDeployment(dep.Name, false); err != nil {
		return err
	}

	log.Info(" Recreating original revision...")
	dep.ResourceVersion = ""
	if len(dep.Annotations) == 0 {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations["nocalhost-dep-ignore"] = "true"
	dep.Annotations[_const.NocalhostApplicationName] = d.AppName
	dep.Annotations[_const.NocalhostApplicationNamespace] = d.NameSpace

	// Add labels and annotations
	if dep.Labels == nil {
		dep.Labels = make(map[string]string, 0)
	}
	dep.Labels[_const.AppManagedByLabel] = _const.AppManagedByNocalhost

	if _, err = clientUtils.CreateDeployment(dep); err != nil {
		if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
			log.Warn("[Warning] Nocalhost-dep needs to update")
		}
		return err
	}
	return nil
}

func GetDefaultPodName(ctx context.Context, p pod_controller.PodController) (string, error) {
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
