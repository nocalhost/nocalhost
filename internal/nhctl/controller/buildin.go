/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"errors"
	"nocalhost/internal/nhctl/common/base"
	profile2 "nocalhost/internal/nhctl/profile"
)

type DevModeAction struct {
	ScaleAction     []profile2.PatchItem
	PodTemplatePath string
	Create          bool
	//Group       string
	//Version     string
	//Kind        string
}

var (
	DeploymentDevModeAction = DevModeAction{
		ScaleAction: []profile2.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template",
	}

	StatefulSetDevModeAction = DevModeAction{
		ScaleAction: []profile2.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template",
	}

	DaemonSetDevModeAction = DevModeAction{
		ScaleAction: []profile2.PatchItem{{
			Patch: `{"spec":{"template": {"spec": {"nodeName": "nocalhost.unreachable"}}}}`,
			Type:  "strategic",
		}},
		PodTemplatePath: "/spec/template",
		Create:          true,
	}
)

func GetDevModeActionBySvcType(svcType base.SvcType) (DevModeAction, error) {
	switch svcType {
	case base.Deployment:
		return DeploymentDevModeAction, nil
	case base.StatefulSet:
		return StatefulSetDevModeAction, nil
	case base.DaemonSet:
		return DaemonSetDevModeAction, nil
		//case base.Job:
		//	return DeploymentDevModeAction
		//case base.CronJob:
		//	return DeploymentDevModeAction
		//case base.Pod:
		//	return DeploymentDevModeAction
	}
	return DeploymentDevModeAction, errors.New("un supported workload")
}

//func (d *DevModeAction) GetResourceType() (string, error) {
//	if d.Kind == "" {
//		return "", errors.New("Resource Kind can not nil")
//	}
//
//	resourceType := d.Kind
//	if d.Version != "" {
//		resourceType += "." + d.Version
//		if d.Group != "" {
//			resourceType += "." + d.Group
//		}
//	}
//	return resourceType, nil
//}
