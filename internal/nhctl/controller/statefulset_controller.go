/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
	"strings"
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

	if err = s.Client.ScaleStatefulSetReplicasToOne(s.GetName()); err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		s.genContainersAndVolumes(&dep.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&dep.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)
	//needToRemovePriorityClass := false
	//for i := 0; i < 10; i++ {
	//	events, err := s.Client.ListEventsByStatefulSet(s.GetName())
	//	utils.Should(err)
	//	_ = s.Client.DeleteEvents(events, true)
	//
	//	// Get the latest stateful set
	//	dep, err = s.Client.GetStatefulSet(s.GetName())
	//	if err != nil {
	//		return err
	//	}
	//
	//	patchDevContainerToPodSpec(&dep.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)
	//
	//	if len(dep.Annotations) == 0 {
	//		dep.Annotations = make(map[string]string, 0)
	//	}
	//	dep.Annotations[OriginSpecJson] = string(originalSpecJson)
	//
	//	log.Info("Updating development container...")
	//
	//	_, err = s.Client.UpdateStatefulSet(dep, true)
	//	if err != nil {
	//		if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
	//			log.Warn("StatefulSet has been modified, retrying...")
	//			continue
	//		}
	//		return err
	//	} else {
	//		// Check if priorityClass exists
	//	outer:
	//		for i := 0; i < 20; i++ {
	//			time.Sleep(1 * time.Second)
	//			events, err = s.Client.ListEventsByStatefulSet(s.GetName())
	//			for _, event := range events {
	//				if strings.Contains(event.Message, "no PriorityClass") {
	//					log.Warn("PriorityClass not found, disable it...")
	//					needToRemovePriorityClass = true
	//					break outer
	//				} else if event.Reason == "SuccessfulCreate" {
	//					log.Infof("Pod SuccessfulCreate")
	//					break outer
	//				}
	//			}
	//		}
	//
	//		if needToRemovePriorityClass {
	//			dep, err = s.Client.GetStatefulSet(s.GetName())
	//			if err != nil {
	//				return err
	//			}
	//			dep.Spec.Template.Spec.PriorityClassName = ""
	//			log.Info("Removing priorityClass")
	//			_, err = s.Client.UpdateStatefulSet(dep, true)
	//			if err != nil {
	//				if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
	//					log.Warn("StatefulSet has been modified, retrying...")
	//					continue
	//				}
	//				return err
	//			}
	//			break
	//		}
	//	}
	//	break
	//}

	ps := genDevContainerPatches(&dep.Spec.Template.Spec, string(originalSpecJson))
	if err := s.Client.Patch(s.Type.String(), dep.Name, ps, "json"); err != nil {
		return err
	}

	s.patchAfterDevContainerReplaced(ops.Container, dep.Kind, dep.Name)

	return waitingPodToBeReady(s.GetNocalhostDevContainerPod)
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
