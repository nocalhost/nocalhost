/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package hub

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/yaml"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
)

func FindNocalhostSvcConfig(appName, svcName string, svcType base.SvcType, container, image string) (*profile.ServiceConfigV2, error) {
	if image == "" {
		return nil, errors.New("Image can not be empty")
	}
	log.Logf("Finding %s's config.yaml by app name ...", appName)
	config, err := findNocalhostConfigByAppName(appName)
	if err != nil {
		log.LogE(err)
	} else {
		svcConfig := config.FindSvcConfigInHub(svcName, svcType, container, image)
		if svcConfig != nil {
			return svcConfig, nil
		}
	}

	log.Log("Finding config in all dir ...")
	return findNocalhostSvcConfigInAllDir(svcName, svcType, container, image)
}

func findNocalhostSvcConfigInAllDir(svcName string, svcType base.SvcType, container, image string) (*profile.ServiceConfigV2, error) {
	svcConfig, err := findNocalhostSvcConfigInIncubator(svcName, svcType, container, image)
	if err != nil {
		log.Log("Failed to find svcConfig in incubator, try to find it from stable")
	} else {
		return svcConfig, nil
	}

	return findNocalhostSvcConfigInStable(svcName, svcType, container, image)
}

func findNocalhostSvcConfigInIncubator(svcName string, svcType base.SvcType, container, image string) (*profile.ServiceConfigV2, error) {
	return findNocalhostSvcConfigInDir(nocalhost_path.GetNocalhostIncubatorHubDir(), svcName, svcType, container, image)
}

func findNocalhostSvcConfigInStable(svcName string, svcType base.SvcType, container, image string) (*profile.ServiceConfigV2, error) {
	return findNocalhostSvcConfigInDir(nocalhost_path.GetNocalhostStableHubDir(), svcName, svcType, container, image)
}

func findNocalhostSvcConfigInDir(dir, svcName string, svcType base.SvcType, container, image string) (*profile.ServiceConfigV2, error) {
	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	for _, d := range dirs {
		if d.IsDir() {
			config, err := findNocalhostConfigByDir(filepath.Join(dir, d.Name()))
			if err != nil {
				continue
			}
			svcConfig := config.FindSvcConfigInHub(svcName, svcType, container, image)
			if svcConfig != nil {
				return svcConfig, nil
			}
		}
	}
	return nil, errors.New("Service config not found")
}

func findNocalhostConfigByAppName(appName string) (*profile.NocalHostAppConfigV2, error) {
	config, err := findNocalhostConfigByDirNameAndAppName("incubator", appName)
	if err != nil {
		log.LogE(err)
	} else {
		return config, err
	}
	return findNocalhostConfigByDirNameAndAppName("stable", appName)
}

func findNocalhostConfigByDirNameAndAppName(dirName, appName string) (*profile.NocalHostAppConfigV2, error) {
	hubDir := nocalhost_path.GetNocalhostHubDir()
	appConfigDirPath := filepath.Join(hubDir, dirName, appName)
	return findNocalhostConfigByDir(appConfigDirPath)
}

// Find a config.yaml in dir dir
func findNocalhostConfigByDir(dir string) (*profile.NocalHostAppConfigV2, error) {
	appConfigDirPath := dir
	s, err := os.Stat(appConfigDirPath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to stat %s", appConfigDirPath))
	}
	if s == nil || !s.IsDir() {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to get %s", appConfigDirPath))
	}

	configPath := filepath.Join(appConfigDirPath, "config.yaml")
	bys, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	configV2 := &profile.NocalHostAppConfigV2{}
	err = yaml.Unmarshal(bys, configV2)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return configV2, nil
}
