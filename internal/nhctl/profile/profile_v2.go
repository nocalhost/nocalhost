/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package profile

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v2"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"strings"
)

const (
	// codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	DefaultDevImage = ""
	DefaultWorkDir  = "/home/nocalhost-dev"
)

type AppProfileV2 struct {
	Name string `json:"name" yaml:"name"`

	// install/uninstall field, should move out of profile
	// Deprecated
	ChartName string `json:"chartName" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	// Deprecated TODO move to appMeta
	ReleaseName string `json:"releaseName yaml:releaseName"`
	// Deprecated
	DependencyConfigMapName string `json:"dependencyConfigMapName" yaml:"dependencyConfigMapName,omitempty"`
	// Deprecated
	Installed bool `json:"installed" yaml:"installed"`
	// Deprecated
	ResourcePath RelPath `json:"resourcePath" yaml:"resourcePath"`
	// Deprecated
	IgnoredPath RelPath `json:"ignoredPath" yaml:"ignoredPath"`
	// Deprecated
	PreInstall SortedRelPath `json:"onPreInstall" yaml:"onPreInstall"`
	// Deprecated
	AppType string `json:"appType" yaml:"appType"`
	// Deprecated
	Env []*Env `json:"env" yaml:"env"`
	// Deprecated
	EnvFrom EnvFrom `json:"envFrom" yaml:"envFrom"`

	// app global field
	Namespace  string `json:"namespace" yaml:"namespace"`
	Kubeconfig string `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	db         *dbutils.LevelDBUtils

	// app global status
	Identifier string `json:"identifier" yaml:"identifier"`
	Secreted   bool   `json:"secreted" yaml:"secreted"` // always true for new versions, but from earlier version, the flag for upload profile to secret

	// svc runtime status
	SvcProfile []*SvcProfileV2 `json:"svcProfile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil

	dbPath  string
	appName string
	ns      string
}

func (a *AppProfileV2) LoadManifests(tmpDir string) ([]string, []string) {
	preInstallManifests := a.PreInstall.Load(tmpDir)
	allManifests := a.ResourcePath.Load(tmpDir)
	ignore := a.IgnoredPath.Load(tmpDir)

	return preInstallManifests, clientgoutils.LoadValidManifest(allManifests, append(preInstallManifests, ignore...))
}

func ProfileV2Key(ns, app string) string {
	return fmt.Sprintf("%s.%s.profile.v2", ns, app)
}

func NewAppProfileV2ForUpdate(ns, name string) (*AppProfileV2, error) {
	path := nocalhost_path.GetAppDbDir(ns, name)
	db, err := dbutils.OpenLevelDB(path, false)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	result := &AppProfileV2{}
	bys, err := db.Get([]byte(ProfileV2Key(ns, name)))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			result, err := db.ListAll()
			if err != nil {
				_ = db.Close()
				return nil, err
			}
			for key, val := range result {
				if strings.Contains(key, "profile.v2") {
					bys = []byte(val)
					break
				}
			}
		} else {
			_ = db.Close()
			return nil, errors.Wrap(err, "")
		}
	}
	if len(bys) == 0 {
		_ = db.Close()
		return nil, errors.New(fmt.Sprintf("Profile not found %s-%s", ns, name))
	}

	err = yaml.Unmarshal(bys, result)
	if err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "")
	}

	result.ns = ns
	result.appName = name
	result.db = db
	result.dbPath = path
	return result, nil
}

func (a *AppProfileV2) SvcProfileV2(svcName string, svcType string) *SvcProfileV2 {

	for _, svcProfile := range a.SvcProfile {
		if svcProfile.ActualName == svcName && svcProfile.Type == svcType {
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
			Type: svcType,
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

// this method will not save the Identifier,
// make sure it will be saving while use
func (a *AppProfileV2) GenerateIdentifierIfNeeded() string {
	if a.Identifier == "" && a != nil {
		u, _ := uuid.NewRandom()
		a.Identifier = u.String()
	}
	return a.Identifier
}

func (a *AppProfileV2) Save() error {
	if a.db == nil {
		return nil
	}
	bys, err := yaml.Marshal(a)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if _, err = os.Stat(a.dbPath); err != nil {
		return errors.Wrap(err, "")
	}
	return a.db.Put([]byte(ProfileV2Key(a.ns, a.appName)), bys)
}

func (a *AppProfileV2) CloseDb() error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}

type SvcProfileV2 struct {
	*ServiceConfigV2 `json:"rawConfig" yaml:"rawConfig"`
	ContainerProfile []*ContainerProfileV2 `json:"containerProfile" yaml:"containerProfile"`
	ActualName       string                `json:"actualName" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name

	PortForwarded bool     `json:"portForwarded" yaml:"portForwarded"`
	Syncing       bool     `json:"syncing" yaml:"syncing"`
	SyncDirs      []string `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
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

	// nocalhost supports config from local dir under "Associate" Path
	// but only load once, and user can use nhctl config reload to reload
	// config from local or secret
	LocalConfigLoaded bool `json:"localconfigloaded" yaml:"localconfigloaded"`

	// associate for the local dir
	Associate string `json:"associate" yaml:"associate"`

	// from app meta
	Developing bool `json:"developing" yaml:"developing"`

	// mean the current controller is possess by current nhctl context
	// and the syncthing process is listen on current device
	Possess bool `json:"possess" yaml:"possess"`
}

type ContainerProfileV2 struct {
	Name string
}

type DevPortForward struct {
	LocalPort         int    `json:"localport" yaml:"localport"`
	RemotePort        int    `json:"remoteport" yaml:"remoteport"`
	Role              string `json:"role" yaml:"role"`
	Status            string `json:"status" yaml:"status"`
	Reason            string `json:"reason" yaml:"reason"`
	PodName           string `json:"podName" yaml:"podName"`
	Updated           string `json:"updated" yaml:"updated"`
	Pid               int    `json:"pid" yaml:"pid"`
	RunByDaemonServer bool   `json:"runByDaemonServer" yaml:"runByDaemonServer"`
	Sudo              bool   `json:"sudo" yaml:"sudo"`
	DaemonServerPid   int    `json:"daemonserverpid" yaml:"daemonserverpid"`
	ServiceType       string `json:"servicetype" yaml:"servicetype"`
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
