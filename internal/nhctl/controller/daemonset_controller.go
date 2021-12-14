/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
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
	return findDevPod(checkPodsList.Items)
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
	return fmt.Sprintf("%s%s", daemonSetGenDeployPrefix, d.GetName())
}

// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
// but create a deployment with dev container instead
func (d *DaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	d.Client.Context(ctx)
	ds, err := d.Client.GetDaemonSet(d.GetName())
	if err != nil {
		return err
	}

	// Scale pod to 0
	err = scaleDaemonSetReplicasToZero(d.GetName(), d.Client)
	if err != nil {
		return err
	}

	// Create a deployment from DaemonSet spec
	generatedDeployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.getGeneratedDeploymentName(),
			Labels: map[string]string{_const.DevWorkloadIgnored: "true"},
		},
		Spec: v1.DeploymentSpec{
			Selector: ds.Spec.Selector,
			Template: ds.Spec.Template,
		},
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(&generatedDeployment.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&generatedDeployment.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	// Create generated deployment
	if _, err = d.Client.CreateDeployment(generatedDeployment); err != nil {
		return err
	}

	d.patchAfterDevContainerReplaced(ops.Container, generatedDeployment.Kind, generatedDeployment.Name)

	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)
}

func (d *DaemonSetController) RollBack(reset bool) error {
	// Delete generated Deployment
	err := d.Client.DeleteDeployment(d.getGeneratedDeploymentName(), false)
	if err != nil {
		return err
	}

	// Remove nodeName in pod spec
	ds, err := d.Client.GetDaemonSet(d.GetName())
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
	if d.IsInReplaceDevMode() {
		return d.Client.ListLatestRevisionPodsByDeployment(d.getGeneratedDeploymentName())
	}
	return d.Client.ListPodsByDaemonSet(d.GetName())
}
