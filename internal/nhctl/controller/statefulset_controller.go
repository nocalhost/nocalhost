/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/model"
)

const (
	OriginSpecJson = "nocalhost.origin.spec.json" // deprecated
)

type StatefulSetController struct {
	*Controller
}

func (s *StatefulSetController) GetNocalhostDevContainerPod() (string, error) {
	//checkPodsList, err := s.Client.ListPodsByStatefulSet(s.GetName())
	//if err != nil {
	//	return "", err
	//}
	ps, err := s.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPodName(ps)
}

func (s *StatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return s.PatchDevModeManifest(ctx, ops)
}

func (s *StatefulSetController) RollBack(reset bool) error {
	return s.RollbackFromAnnotation()
	//clientUtils := s.Client
	//
	//dep, err := clientUtils.GetStatefulSet(s.GetName())
	//if err != nil {
	//	return err
	//}
	//
	//osj, ok := dep.Annotations[OriginSpecJson]
	//if !ok {
	//	return errors.New("No spec json found to rollback")
	//}
	//
	//log.Info(" Deleting current revision...")
	//if err = clientUtils.DeleteStatefulSet(dep.Name); err != nil {
	//	return err
	//}
	//
	//dep.Spec = v1.StatefulSetSpec{}
	//if err = json.Unmarshal([]byte(osj), &dep.Spec); err != nil {
	//	return errors.Wrap(err, "")
	//}
	//
	//log.Info(" Recreating original revision...")
	//dep.ResourceVersion = ""
	//if len(dep.Annotations) == 0 {
	//	dep.Annotations = make(map[string]string, 0)
	//}
	//dep.Annotations["nocalhost-dep-ignore"] = "true"
	//dep.Annotations[OriginSpecJson] = osj
	//
	//// Add labels and annotations
	//if dep.Labels == nil {
	//	dep.Labels = make(map[string]string, 0)
	//}
	//dep.Labels[_const.AppManagedByLabel] = _const.AppManagedByNocalhost
	//
	//if dep.Annotations == nil {
	//	dep.Annotations = make(map[string]string, 0)
	//}
	//dep.Annotations[_const.NocalhostApplicationName] = s.AppName
	//dep.Annotations[_const.NocalhostApplicationNamespace] = s.NameSpace
	//
	//_, err = clientUtils.CreateStatefulSet(dep)
	//if err != nil {
	//	if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
	//		log.Warn("[Warning] Nocalhost-dep needs to update")
	//	}
	//	return err
	//}
	//return nil
}

func (s *StatefulSetController) GetPodList() ([]corev1.Pod, error) {
	//list, err := s.Client.ListPodsByStatefulSet(s.GetName())
	//if err != nil {
	//	return nil, err
	//}
	//if list == nil || len(list.Items) == 0 {
	//	return nil, errors.New("no pod found")
	//}
	//return list.Items, nil
	return s.Controller.GetPodList()
}
