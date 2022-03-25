/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

func GetAllKubeconfig() ([]string, error) {
	apps, err := GetNsAndApplicationInfo(false, false)
	if err != nil {
		return nil, err
	}
	kubeMap := make(map[string]string, 0)
	for _, app := range apps {
		p, err := GetProfileV2(app.Namespace, app.Name, app.Nid)
		if err != nil {
			continue
		}
		if p.Kubeconfig != "" {
			kubeMap[p.Kubeconfig] = ""
		}
	}
	result := make([]string, 0)
	for s, _ := range kubeMap {
		result = append(result, s)
	}
	return result, nil
}
