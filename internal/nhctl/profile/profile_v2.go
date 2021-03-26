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

package profile

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v2"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
)

const (
	DefaultDevImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	DefaultWorkDir  = "/home/nocalhost-dev"
)

type AppProfileV2 struct {
	Name                    string            `json:"name" yaml:"name"`
	ChartName               string            `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string            `json:"release_name yaml:releaseName"`
	Namespace               string            `json:"namespace" yaml:"namespace"`
	Kubeconfig              string            `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string            `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 string            `json:"app_type" yaml:"appType"`
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
	db      *leveldb.DB
	appName string
	ns      string
}

func ProfileV2Key(ns, app string) string {
	return fmt.Sprintf("%s.%s.profile.v2", ns, app)
}

func OpenApplicationLevelDB(ns, app string, readonly bool) (*leveldb.DB, error) {
	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.OpenApplicationLevelDB(path, readonly)
}

func NewAppProfileV2(ns, name string, readonly bool) (*AppProfileV2, error) {
	db, err := OpenApplicationLevelDB(ns, name, readonly)
	if err != nil {
		return nil, err
	}

	result := &AppProfileV2{}
	bys, err := db.Get([]byte(ProfileV2Key(ns, name)), nil)
	if err != nil {
		db.Close()
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "")
	}
	if len(bys) == 0 {
		db.Close()
		return nil, nil
	}

	err = yaml.Unmarshal(bys, result)
	if err != nil {
		db.Close()
		return nil, errors.Wrap(err, "")
	}

	result.ns = ns
	result.appName = name
	result.db = db
	return result, nil
}

func (a *AppProfileV2) FetchSvcProfileV2FromProfile(svcName string) *SvcProfileV2 {

	for _, svcProfile := range a.SvcProfile {
		if svcProfile.ActualName == svcName {
			return svcProfile
		}
	}

	// If not profile found, init one
	if a.SvcProfile == nil {
		a.SvcProfile = make([]*SvcProfileV2, 0)
	}
	svcProfile := &SvcProfileV2{
		ServiceConfigV2: &ServiceConfigV2{
			Name: svcName,
			Type: string("Deployment"),
			ContainerConfigs: []*ContainerConfig{
				{
					Dev: &ContainerDevConfig{
						Image:   DefaultDevImage,
						WorkDir: DefaultWorkDir,
					},
				},
			},
		},
		ActualName: svcName,
	}
	a.SvcProfile = append(a.SvcProfile, svcProfile)

	return svcProfile
}

func (a *AppProfileV2) Save() error {
	//defer a.db.Close()
	bys, err := yaml.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "")
	}

	//log.Infof("Saving %v", *a)
	//log.Infof("pf:")
	//for _, pf := range a.FetchSvcProfileV2FromProfile("productpage").DevPortForwardList {
	//	log.Infof("%v", *pf)
	//}
	return errors.Wrap(a.db.Put([]byte(ProfileV2Key(a.ns, a.appName)), bys, nil), "")
}

func (a *AppProfileV2) CloseDb() error {
	return a.db.Close()
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
	//DevPortList           []string `json:"devPortList" yaml:"devPortList"`
	//PortForwardStatusList []string `json:"portForwardStatusList" yaml:"portForwardStatusList"`
	//PortForwardPidList    []string `json:"portForwardPidList" yaml:"portForwardPidList"`
}

type ContainerProfileV2 struct {
	Name string
}

type DevPortForward struct {
	LocalPort         int
	RemotePort        int
	Way               string
	Status            string
	Reason            string
	PodName           string `json:"podName" yaml:"podName"`
	Updated           string
	Pid               int
	RunByDaemonServer bool `json:"runByDaemonServer" yaml:"runByDaemonServer"`
	Sudo              bool `json:"sudo"`
	DaemonServerPid   int  `json:"daemonServerPid"`
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
	if len(s.ContainerConfigs) == 0 {
		return nil
	}
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
