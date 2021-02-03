/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"nocalhost/pkg/nhctl/log"
	"os"
)

type AppProfileV2 struct {
	Name                    string          `json:"name" yaml:"name"`
	ChartName               string          `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string          `json:"release_name yaml:releaseName"`
	Namespace               string          `json:"namespace" yaml:"namespace"`
	Kubeconfig              string          `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string          `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType         `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfileV2 `json:"svc_profile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool            `json:"installed" yaml:"installed"`
	ResourcePath            []string        `json:"resource_path" yaml:"resourcePath"`
	IgnoredPath             []string        `json:"ignoredPath" yaml:"ignoredPath"`
}

type ContainerProfileV2 struct {
	Name string
}

type SvcProfileV2 struct {
	*ServiceConfigV2 `yaml:"rawConfig"`
	ContainerProfile []*ContainerProfileV2 `json:"container_profile" yaml:"containerProfile"`
	ActualName       string                `json:"actual_name" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	Developing       bool                  `json:"developing" yaml:"developing"`
	PortForwarded    bool                  `json:"port_forwarded" yaml:"portForwarded"`
	Syncing          bool                  `json:"syncing" yaml:"syncing"`
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int    `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	SyncthingSecret        string `json:"syncthingSecret" yaml:"syncthingSecret"` // secret name
	// syncthing local port
	LocalSyncthingPort                     int      `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int      `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `json:"devPortList" yaml:"devPortList"`
	PortForwardStatusList                  []string `json:"portForwardStatusList" yaml:"portForwardStatusList"`
	PortForwardPidList                     []string `json:"portForwardPidList" yaml:"portForwardPidList"`
}

func (a *Application) LoadAppProfileV2() error {
	app := &AppProfileV2{}

	isV2, err := a.checkIfAppProfileIsV2()
	if err != nil {
		return err
	}

	if !isV2 {
		log.Info("Upgrade profile V1 to V2 ...")
		err = a.UpgradeAppProfileV1ToV2()
		if err != nil {
			return err
		}
	}

	fBytes, err := ioutil.ReadFile(a.getProfileV2Path())
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = yaml.Unmarshal(fBytes, app)
	if err != nil {
		errors.Wrap(err, "")
	}
	a.AppProfileV2 = app
	return nil
}

func (a *Application) checkIfAppProfileIsV2() (bool, error) {
	_, err := os.Stat(a.getProfileV2Path())
	if err == nil {
		return true, nil
	}

	if !os.IsNotExist(err) {
		return false, errors.Wrap(err, "")
	}
	return false, nil
}

func (a *Application) UpgradeAppProfileV1ToV2() error {
	return ConvertAppProfileFileV1ToV2(a.getProfilePath(), a.getProfileV2Path())
}
