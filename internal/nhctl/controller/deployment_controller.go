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
	"nocalhost/pkg/nhctl/log"
	"strings"
	"time"
)

type DeploymentController struct {
	*Controller
}

// ReplaceImage In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (d *DeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return d.PatchDevModeManifest(ctx, ops)
}

func (d *DeploymentController) RollBack(reset bool) error {

	clientUtils := d.Client
	dep, err := d.Client.GetDeployment(d.Name)
	if err != nil {
		return err
	}

	if err = d.RollbackFromAnnotation(); err == nil {
		return nil
	}

	log.Warn(err.Error())

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

func GetDefaultPodName(ctx context.Context, p *Controller) (string, error) {
	var (
		podList []corev1.Pod
		err     error
	)
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", p.Name))
		default:
			podList, err = p.GetPodList()
		}
		if err != nil || len(podList) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", p.Name)
			time.Sleep(time.Second)
		} else {
			return podList[0].Name, nil
		}
	}
}
