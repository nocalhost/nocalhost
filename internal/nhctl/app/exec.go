/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) Exec(svcName string, container string, commands []string) error {
	podList, err := a.client.ListPodsByDeployment(svcName)
	if err != nil {
		return err
	}
	var runningPod = make([]v1.Pod, 0, 1)
	for _, item := range podList.Items {
		if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
			runningPod = append(runningPod, item)
		}
	}
	if len(runningPod) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := runningPod[0].Name
	var name string
	// if container arguments are available, using container arguments
	// else if found nocalhost-dev container, using nocalhost-dev
	// else return error
	for _, c := range runningPod[0].Spec.Containers {
		if c.Name == "nocalhost-dev" {
			name = c.Name
		}
		if c.Name == container {
			name = container
			break
		}
	}
	return a.client.Exec(pod, name, commands)
}
