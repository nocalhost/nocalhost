/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package utils

import (
	"io"
	"net/http"
	"nocalhost/internal/nocalhost-api/global"
	"sigs.k8s.io/yaml"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/repo"
)

type IndexFile struct {
	repo.IndexFile
}

func (i IndexFile) getVersionList(name string) []string {
	cv := i.Entries[name]
	ver := make([]string, 0)
	for _, c := range cv {
		if strings.Contains(c.Version, "-") {
			continue
		}
		ver = append(ver, c.Version)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ver)))
	return ver
}

func GetVClusterVersionList(repoURL string) []string {
	if repoURL == "" {
		repoURL = global.NocalhostChartRepository
	}

	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := c.Get(repoURL + "/index.yaml")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	i := &IndexFile{}
	if err := yaml.UnmarshalStrict(body, i); err != nil {
		return nil
	}
	return i.getVersionList("vcluster")
}
