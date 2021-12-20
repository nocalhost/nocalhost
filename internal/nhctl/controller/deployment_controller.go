/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/pod_controller"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

type DeploymentController struct {
	*Controller
}

func (d *DeploymentController) GetNocalhostDevContainerPod() (string, error) {
	return d.GetDevModePodName()
}

// ReplaceImage In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (d *DeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return d.PatchDevModeManifest(ctx, ops)
}

func (d *DeploymentController) RollBack(reset bool) error {

	clientUtils := d.Client
	dep, err := d.Client.GetDeployment(d.GetName())
	if err != nil {
		return err
	}

	if err = d.RollbackFromAnnotation(); err == nil {
		return nil
	}

	log.Warn(err.Error())
	log.Info("Annotation nocalhost.origin.spec.json not found, try to find it from rs")
	rss, _ := clientUtils.GetSortedReplicaSetsByDeployment(d.GetName())
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
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", p.GetName()))
		default:
			podList, err = p.GetPodList()
		}
		if err != nil || len(podList) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", p.GetName())
			time.Sleep(time.Second)
		} else {
			return podList[0].Name, nil
		}
	}
}

func (d *DeploymentController) GetPodList() ([]corev1.Pod, error) {
	//return d.Client.ListLatestRevisionPodsByDeployment(d.GetName())
	return d.Controller.GetPodList()
}
