/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/client-go/rest"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

const defaultValues = `
storage:
  size: 10Gi
syncer:
  extraArgs: ["--disable-sync-resources=ingresses"]
  resources:
    limits:
      cpu: "1"
      memory: 1Gi
vcluster:
  resources:
    limits:
      cpu: "2"
      memory: 2Gi
`

func ExtraValues(config *rest.Config, vc *helmv1alpha1.VirtualCluster) (map[string]interface{}, error) {
	cidr := getCIDR(config, vc.GetNamespace())
	rel, err := getDefaultValues()
	if err != nil {
		return nil, err
	}
	extraVals := fmt.Sprintf("vcluster.extraArgs={--service-cidr=%s}", cidr)
	svcType := vc.GetServiceType()
	if svcType != "" {
		svcVal := fmt.Sprintf("service.type=%s", svcType)
		err = strvals.ParseInto(svcVal, rel)
		if err != nil {
			return nil, err
		}
	}
	err = strvals.ParseInto(extraVals, rel)
	return rel, err
}

func getDefaultValues() (map[string]interface{}, error) {
	return chartutil.ReadValues([]byte(defaultValues))
}
