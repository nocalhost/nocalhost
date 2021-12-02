/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import profile2 "nocalhost/internal/nhctl/profile"

type DevModeAction struct {
	ScaleAction []profile2.PatchItem
	PodSpecPath string
	Group       string
	Version     string
	Kind        string
}

var (
	DeploymentDevModeAction = DevModeAction{
		ScaleAction: []profile2.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodSpecPath: "/spec/template",
		Kind:        "Deployment",
	}

	StatefulSetDevModeAction = DevModeAction{
		ScaleAction: []profile2.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodSpecPath: "/spec/template",
		Kind:        "StatefulSet",
	}
)
