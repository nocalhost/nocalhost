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
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

type DaemonSetController struct {
	*Controller
}

const daemonSetGenDeployPrefix = "daemon-set-generated-deploy-"

func (d *DaemonSetController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := d.Client.ListPodsByDeployment(d.getGeneratedDeploymentName())
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList)
}

func scaleDaemonSetReplicasToZero(name string, client *clientgoutils.ClientGoUtils) error {

	// Scale pod to 0
	ds, err := client.GetDaemonSet(name)
	if err != nil {
		return err
	}

	ds.Spec.Template.Spec.NodeName = "nocalhost.unreachable"
	if _, err = client.UpdateDaemonSet(ds); err != nil {
		return err
	}

	log.Info("Wait replicas scaling to 0 (timeout: 5min)")
	sp := utils.NewSpinner(fmt.Sprintf("Replicas is %d now", ds.Status.CurrentNumberScheduled))
	sp.Start()
	for i := 0; i < 300; i++ {
		time.Sleep(1 * time.Second)
		ds, err = client.GetDaemonSet(name)
		if err != nil {
			return err
		}
		if ds.Status.CurrentNumberScheduled != 0 {
			sp.Update(fmt.Sprintf("Replicas is %d now", ds.Status.CurrentNumberScheduled))
		} else {
			sp.Update("Replicas has been scaled to 0")
			sp.Stop()
			break
		}
	}
	return nil
}

func (d *DaemonSetController) getGeneratedDeploymentName() string {
	return fmt.Sprintf("%s%s", daemonSetGenDeployPrefix, d.Name())
}

// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
// but create a deployment with dev container instead
func (d *DaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	d.Client.Context(ctx)
	ds, err := d.Client.GetDaemonSet(d.Name())
	if err != nil {
		return err
	}

	// Scale pod to 0
	err = scaleDaemonSetReplicasToZero(d.Name(), d.Client)
	if err != nil {
		return err
	}

	// Create a deployment from DaemonSet spec
	generatedDeployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.getGeneratedDeploymentName(),
			Labels: map[string]string{nocalhost.DevWorkloadIgnored: "true"},
		},
		Spec: v1.DeploymentSpec{
			Selector: ds.Spec.Selector,
			Template: ds.Spec.Template,
		},
	}

	devContainer, err := findContainerInDeploySpec(generatedDeployment, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(devContainer, ops.Container, ops.StorageClass)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range generatedDeployment.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				generatedDeployment.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		generatedDeployment.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if generatedDeployment.Spec.Template.Spec.Volumes == nil {
		generatedDeployment.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	generatedDeployment.Spec.Template.Spec.Volumes = append(generatedDeployment.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	generatedDeployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(generatedDeployment.Spec.Template.Spec.Containers); i++ {
		generatedDeployment.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].StartupProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].SecurityContext = nil
	}

	generatedDeployment.Spec.Template.Spec.Containers =
		append(generatedDeployment.Spec.Template.Spec.Containers, *sideCarContainer)

	// Create generated deployment
	if _, err = d.Client.CreateDeployment(generatedDeployment); err != nil {
		return err
	}

	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)
}

func (d *DaemonSetController) Name() string {
	return d.Controller.Name
}

func (d *DaemonSetController) RollBack(reset bool) error {
	// Delete generated Deployment
	err := d.Client.DeleteDeployment(d.getGeneratedDeploymentName(), false)
	if err != nil {
		return err
	}

	// Remove nodeName in pod spec
	ds, err := d.Client.GetDaemonSet(d.Name())
	if err != nil {
		return err
	}

	ds.Spec.Template.Spec.NodeName = ""
	_, err = d.Client.UpdateDaemonSet(ds)
	return err
}

// GetPodList
// In DevMode, return pod list of generated Deployment.
// Otherwise, return pod list of DaemonSet
func (d *DaemonSetController) GetPodList() ([]corev1.Pod, error) {
	if d.IsInDevMode() {
		return d.Client.ListLatestRevisionPodsByDeployment(d.getGeneratedDeploymentName())
	}
	return d.Client.ListPodsByDaemonSet(d.Name())
}
