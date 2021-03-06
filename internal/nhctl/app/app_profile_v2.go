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
	"strconv"
	"strings"
	"time"
)

type AppProfileV2 struct {
	Name                    string            `json:"name" yaml:"name"`
	ChartName               string            `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string            `json:"release_name yaml:releaseName"`
	Namespace               string            `json:"namespace" yaml:"namespace"`
	Kubeconfig              string            `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string            `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType           `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfileV2   `json:"svc_profile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool              `json:"installed" yaml:"installed"`
	SyncDirs                []string          `json:"syncDirs" yaml:"syncDirs"` // dev start -s
	ResourcePath            []string          `json:"resource_path" yaml:"resourcePath"`
	IgnoredPath             []string          `json:"ignoredPath" yaml:"ignoredPath"`
	PreInstall              []*PreInstallItem `json:"onPreInstall" yaml:"onPreInstall"`

	// After v2
	GitUrl       string `json:"gitUrl" yaml:"gitUrl"`
	GitRef       string `json:"gitRef" yaml:"gitRef"`
	HelmRepoUrl  string `json:"helmRepoUrl" yaml:"helmRepoUrl"`
	HelmRepoName string `json:"helmRepoUrl" yaml:"helmRepoName"`
	//HelmRepoChartVersion string `json:"helmRepoChartVersion" yaml:"helmRepoChartVersion"`

	Env     []*Env  `json:"env" yaml:"env"`
	EnvFrom EnvFrom `json:"envFrom" yaml:"envFrom"`
}

type ContainerProfileV2 struct {
	Name string
}

type DevPortForward struct {
	LocalPort  int
	RemotePort int
	Way        string
	Status     string
	Reason     string
	Updated    string
	Pid        int
}

type SvcProfileV2 struct {
	*ServiceConfigV2 `yaml:"rawConfig"`
	ContainerProfile []*ContainerProfileV2 `json:"container_profile" yaml:"containerProfile"`
	ActualName       string                `json:"actual_name" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	Developing       bool                  `json:"developing" yaml:"developing"`
	PortForwarded    bool                  `json:"port_forwarded" yaml:"portForwarded"`
	Syncing          bool                  `json:"syncing" yaml:"syncing"`
	SyncDirs         []string              `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int    `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	SyncthingSecret        string `json:"syncthingSecret" yaml:"syncthingSecret"` // secret name
	// syncthing local port
	LocalSyncthingPort                     int               `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int               `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string          `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortForwardList                     []*DevPortForward `json:"devPortForwardList" yaml:"devPortForwardList"` // combine DevPortList,PortForwardStatusList and PortForwardPidList
	// Deprecated later
	DevPortList           []string `json:"devPortList" yaml:"devPortList"`
	PortForwardStatusList []string `json:"portForwardStatusList" yaml:"portForwardStatusList"`
	PortForwardPidList    []string `json:"portForwardPidList" yaml:"portForwardPidList"`
}

func (a *Application) convertDevPortForwardList() {
	var err error
	changed := false
	for _, svcProfile := range a.AppProfileV2.SvcProfile {
		if len(svcProfile.DevPortForwardList) > 0 {
			continue // already convert
		}
		for _, portString := range svcProfile.DevPortList {
			log.Debugf("Converting %s", portString)
			changed = true
			devPortForward := &DevPortForward{
				Way:    "",
				Status: "",
			}
			svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, devPortForward)

			ports := strings.Split(portString, ":")
			devPortForward.LocalPort, err = strconv.Atoi(ports[0])
			if err != nil {
				log.WarnE(errors.Wrap(err, ""), err.Error())
			}
			devPortForward.RemotePort, err = strconv.Atoi(ports[1])
			if err != nil {
				log.WarnE(errors.Wrap(err, ""), err.Error())
			}

			// find way and status
			for _, statusString := range svcProfile.PortForwardStatusList {
				if strings.Contains(statusString, portString) {
					// eg: 8091:8091(MANUAL-LISTEN)
					str := strings.Split(statusString, "(") // MANUAL-LISTEN)
					str = strings.Split(str[1], ")")        // MANUAL-LISTEN
					str = strings.Split(str[0], "-")
					devPortForward.Way = str[0]
					devPortForward.Status = str[1]
					log.Debugf("%s's status is %s-%s", devPortForward.Way, devPortForward.Status)
					break
				}
			}
			// find pid
			for _, pidString := range svcProfile.PortForwardPidList {
				if strings.Contains(pidString, portString) {
					// eg: 8091:8091-16768
					pidStr := strings.Split(pidString, "-")[1]
					devPortForward.Pid, err = strconv.Atoi(pidStr)
					if err != nil {
						log.WarnE(errors.Wrap(err, ""), err.Error())
					}
					log.Debugf("%s's pid is %d", pidString, devPortForward.Pid)
					break
				}
			}

		}
		svcProfile.PortForwardPidList = nil
		svcProfile.PortForwardStatusList = nil
		svcProfile.DevPortList = nil
	}
	if changed {
		_ = a.SaveProfile()
	}
}

// Compatible for v1
// Finding `containerName` config, if not found, use the first container config
func (s *SvcProfileV2) GetContainerDevConfigOrDefault(containerName string) *ContainerDevConfig {
	config := s.GetContainerDevConfig(containerName)
	if config == nil {
		config = s.GetDefaultContainerDevConfig()
	}
	return config
}

func (s *SvcProfileV2) GetDefaultContainerDevConfig() *ContainerDevConfig {
	//if s.ContainerConfigs[0].Name == "" {
	//	return s.ContainerConfigs[0].Dev
	//}
	return s.ContainerConfigs[0].Dev
}

func (s *SvcProfileV2) GetContainerDevConfig(containerName string) *ContainerDevConfig {
	for _, devConfig := range s.ContainerConfigs {
		if devConfig.Name == containerName {
			return devConfig.Dev
		}
	}
	return nil
}

func (a *Application) LoadAppProfileV2(retry bool) error {
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

	if app.Namespace == "" && retry {
		log.Warn("Failed to load profile, retry...")
		timeout := true
		for i := 0; i < 100; i++ {
			time.Sleep(1 * time.Second)
			fBytes, err = ioutil.ReadFile(a.getProfileV2Path())
			if err != nil {
				return errors.Wrap(err, "")
			}
			err = yaml.Unmarshal(fBytes, app)
			if err != nil {
				errors.Wrap(err, "")
			}
			if app.Namespace == "" {
				log.Info("Reloading profile failed, try again...")
			} else {
				log.Info("Reloading profile succeeded!")
				timeout = false
				break
			}
		}
		if timeout {
			return errors.New("Failed to load profile after 100 retrying")
		}
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

func (a *Application) checkIfAppConfigIsV2() (bool, error) {
	_, err := os.Stat(a.GetConfigV2Path())
	if err == nil {
		return true, nil
	}

	if !os.IsNotExist(err) {
		return false, errors.Wrap(err, "")
	}
	return false, nil
}

func (a *Application) UpgradeAppProfileV1ToV2() error {
	err := ConvertAppProfileFileV1ToV2(a.getProfilePath(), a.getProfileV2Path())
	if err != nil {
		return err
	}
	return os.Rename(a.getProfilePath(), a.getProfilePath()+".bak")
}

func (a *Application) UpgradeAppConfigV1ToV2() error {
	err := ConvertConfigFileV1ToV2(a.GetConfigPath(), a.GetConfigV2Path())
	if err != nil {
		return err
	}
	return os.Rename(a.GetConfigPath(), a.GetConfigPath()+".bak")
}
