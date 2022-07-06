/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"fmt"
	"io/ioutil"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var supportedSvcType map[base.SvcType]base.DevModeAction
var buildInGvkList []*schema.GroupVersionKind

func IsBuildInGvk(gvk *schema.GroupVersionKind) bool {
	for _, kind := range buildInGvkList {
		if gvk.Kind == kind.Kind && gvk.Version == kind.Version && gvk.Group == kind.Group {
			return true
		}
	}
	return false
}

func init() {
	supportedSvcType = make(map[base.SvcType]base.DevModeAction, 0)
	supportedSvcType[base.Deployment] = DefaultDevModeAction
	supportedSvcType["deployments.v1.apps"] = DefaultDevModeAction
	supportedSvcType["deployments.v1beta1.apps"] = DefaultDevModeAction
	supportedSvcType["deployments.v1beta2.apps"] = DefaultDevModeAction
	supportedSvcType["deployments.v1beta1.extensions"] = DefaultDevModeAction
	supportedSvcType[base.StatefulSet] = DefaultDevModeAction
	supportedSvcType["statefulsets.v1.apps"] = DefaultDevModeAction
	supportedSvcType["statefulsets.v1beta1.apps"] = DefaultDevModeAction
	supportedSvcType["statefulsets.v1beta2.apps"] = DefaultDevModeAction
	supportedSvcType["statefulsets.v1beta1.extensions"] = DefaultDevModeAction
	supportedSvcType[base.DaemonSet] = DaemonSetDevModeAction
	supportedSvcType[base.Job] = JobDevModeAction
	supportedSvcType[base.CronJob] = CronJobDevModeAction
	supportedSvcType[base.CronJob] = CronJobDevModeAction
	supportedSvcType[base.Pod] = PodDevModeAction   // Todo
	supportedSvcType["pods.v1."] = PodDevModeAction // Todo

	// Kruise
	supportedSvcType["clonesets.v1alpha1.apps.kruise.io"] = DefaultDevModeAction
	supportedSvcType["statefulsets.v1beta1.apps.kruise.io"] = DefaultDevModeAction
	supportedSvcType["daemonsets.v1alpha1.apps.kruise.io"] = DaemonSetDevModeAction
	supportedSvcType["advancedcronjobs.v1alpha1.apps.kruise.io"] = KruiseCronJobDevModeAction
	supportedSvcType["broadcastjobs.v1alpha1.apps.kruise.io"] = JobDevModeAction

	buildInGvkList = []*schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "batch", Version: "v1", Kind: "Job"},
		{Group: "batch", Version: "v1", Kind: "CronJob"},
		{Group: "batch", Version: "v1beta1", Kind: "CronJob"},
		{Group: "", Version: "v1", Kind: "Pod"},
	}

	bys, err := ioutil.ReadFile(filepath.Join(nocalhost_path.GetNhctlHomeDir(), "config"))
	if err == nil && len(bys) > 0 {
		configFile := base.ConfigFile{}
		if err = yaml.Unmarshal(bys, &configFile); err != nil {
			log.WarnE(err, "")
			return
		}
		for _, action := range configFile.CrdDevModeActions {
			if action.CrdType != "" && action.DevModeAction != nil {
				supportedSvcType[base.SvcType(action.CrdType)] = *action.DevModeAction
			}
		}
	}
}

func GetDevModeActionBySvcType(svcType base.SvcType) (*base.DevModeAction, error) {
	if da, ok := supportedSvcType[svcType]; ok {
		return &da, nil
	}
	return nil, errors.New(fmt.Sprintf("Workload Type %s is unsupported", svcType))
}

func CheckIfResourceTypeIsSupported(svcType base.SvcType) bool {
	if _, ok := supportedSvcType[svcType]; ok {
		return true
	}
	return false
}

func SvcTypeOfMutate(svcType string) (base.SvcType, error) {
	_, err := GetDevModeActionBySvcType(base.SvcType(svcType))
	return base.SvcType(svcType), err
}

var (
	DefaultDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template",
	}

	PodDevModeAction = base.DevModeAction{}

	DaemonSetDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `[{"op":"replace","path": "/spec/template/spec/nodeName", "value": "nocalhost.unreachable"}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template",
		Create:          true,
	}

	JobDevModeAction = base.DevModeAction{
		PodTemplatePath: "/spec/template",
		Create:          true,
	}

	CronJobDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `{"spec":{"suspend": true}}`,
			Type:  "strategic",
		}},
		PodTemplatePath: "/spec/jobTemplate/spec/template",
		Create:          true,
	}

	KruiseCronJobDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `[{"op":"replace","path": "/spec/suspend", "value": true}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template/broadcastJobTemplate/spec/template",
		Create:          true,
	}
)
