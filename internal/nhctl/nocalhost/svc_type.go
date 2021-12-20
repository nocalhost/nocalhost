/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
)

var supportedSvcType map[base.SvcType]base.DevModeAction
var buildInGvkList []*schema.GroupVersionKind

const (
	Deployment       base.SvcType = "deployment"
	StatefulSet      base.SvcType = "statefulset"
	DaemonSet        base.SvcType = "daemonset"
	Job              base.SvcType = "job"
	CronJob          base.SvcType = "cronjob"
	Pod              base.SvcType = "pod"
	CloneSetV1Alpha1 base.SvcType = "clonesets.v1alpha1.apps.kruise.io"
)

//func SvcTypeIsBuildIn(svcType base.SvcType) bool {
//	if svcType == Deployment || svcType == StatefulSet || svcType == DaemonSet ||
//		svcType == Job || svcType == CronJob || svcType == Pod {
//		return true
//	}
//	return false
//}

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
	supportedSvcType[Deployment] = DefaultDevModeAction
	supportedSvcType[StatefulSet] = StatefulSetDevModeAction
	supportedSvcType[DaemonSet] = DaemonSetDevModeAction
	supportedSvcType[Job] = JobDevModeAction
	supportedSvcType[CronJob] = CronJobDevModeAction
	supportedSvcType[Pod] = DefaultDevModeAction // Todo
	supportedSvcType[CloneSetV1Alpha1] = DefaultDevModeAction

	buildInGvkList = []*schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "batch", Version: "v1", Kind: "Job"},
		{Group: "batch", Version: "v1", Kind: "CronJob"},
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

	StatefulSetDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `[{"op":"replace","path":"/spec/replicas","value":1}]`,
			Type:  "json",
		}},
		PodTemplatePath: "/spec/template",
	}

	DaemonSetDevModeAction = base.DevModeAction{
		ScalePatches: []base.PatchItem{{
			Patch: `{"spec":{"template": {"spec": {"nodeName": "nocalhost.unreachable"}}}}`,
			Type:  "strategic",
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
)
