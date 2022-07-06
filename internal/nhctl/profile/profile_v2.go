/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DevModeType string

const (
	// nocalhost-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	DefaultDevImage  = ""
	DefaultWorkDir   = "/home/nocalhost-dev"
	DuplicateDevMode = DevModeType("duplicate")
	ReplaceDevMode   = DevModeType("replace")
	NoneDevMode      = DevModeType("")
)

func (d DevModeType) IsReplaceDevMode() bool {
	if d == ReplaceDevMode || d == "" {
		return true
	}
	return false
}

func (d DevModeType) IsDuplicateDevMode() bool {
	return d == DuplicateDevMode
}

func (d DevModeType) ToString() string {
	if d == "" {
		return string(ReplaceDevMode)
	}
	return string(d)
}

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
	//Env []*Env `json:"env" yaml:"env"`
	//// Deprecated
	//EnvFrom EnvFrom `json:"envFrom" yaml:"envFrom"`

	// app global field
	Namespace  string `json:"namespace" yaml:"namespace"`
	Kubeconfig string `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	db         *dbutils.LevelDBUtils

	// for previous version, associate path is stored in profile
	// and now it store in a standalone db
	AssociateMigrate bool `json:"associate_migrate" yaml:"associate_migrate"`

	// for previous version, config is stored in profile
	// and now it store in app meta
	//ConfigMigrated bool `json:"configMigrated" yaml:"configMigrated"`

	// app global status
	Identifier string `json:"identifier" yaml:"identifier"`
	Secreted   bool   `json:"secreted" yaml:"secreted"` // always true for new versions, but from earlier version, the flag for upload profile to secret

	// svc runtime status
	SvcProfile []*SvcProfileV2 `json:"svcProfile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil

	dbPath  string
	appName string
	ns      string
}

func ProfileV2Key(ns, app string) string {
	return fmt.Sprintf("%s.%s.profile.v2", ns, app)
}

func NewAppProfileV2ForUpdate(ns, name, nid string) (*AppProfileV2, error) {
	path := nocalhost_path.GetAppDbDir(ns, name, nid)
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

// SvcProfileV2 The result will not be nil
func (a *AppProfileV2) SvcProfileV2(svcName string, svcType string) *SvcProfileV2 {

	for _, svcProfile := range a.SvcProfile {
		if svcProfile.GetName() == svcName && svcProfile.GetType() == svcType {
			return svcProfile
		}
	}

	// If not profile found, init one
	if a.SvcProfile == nil {
		a.SvcProfile = make([]*SvcProfileV2, 0)
	}
	svcProfile := &SvcProfileV2{
		Type: svcType,
		Name: svcName,
	}
	a.SvcProfile = append(a.SvcProfile, svcProfile)

	return svcProfile
}

// this method will not save the Identifier,
// make sure it will be saving while use
func (a *AppProfileV2) GenerateIdentifierIfNeeded() string {
	if a != nil && a.Identifier == "" {
		var s string
		if address := getMacAddress(); address != nil {
			s = address.String()
		} else if random, err := uuid.NewUUID(); err == nil {
			s = random.String()
		} else {
			s = strconv.Itoa(time.Now().Nanosecond())
		}
		a.Identifier = "i" + strings.ReplaceAll(s, ":", "-")
	}
	return a.Identifier
}

func getMacAddress() net.HardwareAddr {
	index, err := net.Interfaces()
	if err == nil {
		maps := make(map[string]net.HardwareAddr, len(index))
		for _, i := range index {
			if i.HardwareAddr != nil && len(i.HardwareAddr) != 0 {
				maps[i.Name] = i.HardwareAddr
			}
		}
		for i := 0; i < len(maps); i++ {
			if addr, ok := maps[fmt.Sprintf("en%v", i)]; ok {
				return addr
			}
		}
		for _, v := range maps {
			return v
		}
	}
	return nil
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
	defer func() {
		for _, profileV2 := range a.SvcProfile {
			if len(profileV2.DevPortForwardList) != 0 {
				ioutil.WriteFile(filepath.Join(filepath.Dir(a.dbPath), "portforward"), nil, 0644)
				break
			}
		}
	}()
	return a.db.Put([]byte(ProfileV2Key(a.ns, a.appName)), bys)
}

func (a *AppProfileV2) CloseDb() error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}

type SvcProfileV2 struct {
	//*ServiceConfigV2 `json:"rawConfig" yaml:"rawConfig"` // deprecated, move to app mate
	//ContainerProfile []*ContainerProfileV2 `json:"containerProfile" yaml:"containerProfile"`
	ActualName string `json:"actualName" yaml:"actualName"` // deprecated - for helm, actualName may be ReleaseName-Name

	// todo: Is this will conflict with ServiceConfigV2 ? by hxx
	Name string `json:"name" yaml:"name"`
	Type string `json:"serviceType" yaml:"serviceType"`

	PortForwarded bool     `json:"portForwarded" yaml:"portForwarded"`
	Syncing       bool     `json:"syncing" yaml:"syncing"`
	SyncDirs      []string `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	//SyncthingSecret                     string `json:"syncthingSecret" yaml:"syncthingSecret"` // secret name
	//DuplicateDevModeSyncthingSecretName string `json:"duplicateDevModeSyncthingSecretName" yaml:"duplicateDevModeSyncthingSecretName"`
	// syncthing local port
	LocalSyncthingPort                     int               `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int               `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string          `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortForwardList                     []*DevPortForward `json:"devPortForwardList" yaml:"devPortForwardList"` // combine DevPortList,PortForwardStatusList and PortForwardPidList

	// nocalhost supports config from local dir under "Associate" Path, it's priority is highest
	LocalConfigLoaded bool `json:"localconfigloaded" yaml:"localconfigloaded"`

	// nocalhost also supports cfg from annotations, it's priority is lower than local cfg and higher than cm cfg
	AnnotationsConfigLoaded bool `json:"annotationsconfigloaded" yaml:"annotationsconfigloaded"`

	// nocalhost also supports config from cm, lowest priority
	CmConfigLoaded bool `json:"cmconfigloaded" yaml:"cmconfigloaded"`

	// deprecated, read only, but actually store in
	// [SvcPack internal/nhctl/nocalhost/dev_dir_mapping_db.go:165]
	// associate for the local dir
	Associate string `json:"associate" yaml:"associate"`

	// deprecated
	// for earlier version of nocalhost
	// from app meta, this status return ture may under start developing (pod not ready, etc..)
	Developing bool `json:"developing" yaml:"developing"`

	// from app meta
	DevelopStatus string `json:"develop_status" yaml:"develop_status"`

	// mean the current controller is possess by current nhctl context
	// and the syncthing process is listen on current device
	Possess bool `json:"possess" yaml:"possess"`

	// LocalDevMode can be started in every local desktop and not influence each other
	DevModeType DevModeType `json:"devModeType" yaml:"devModeType"` // Used by ide plugin

	// recorded the container that enter the devmode
	// notice: exit devmode will not set this value to null
	OriginDevContainer string `json:"originDevContainer" yaml:"originDevContainer"`
}

type ContainerProfileV2 struct {
	Name string
}

type DevPortForward struct {
	LocalPort       int               `json:"localport" yaml:"localport"`
	RemotePort      int               `json:"remoteport" yaml:"remoteport"`
	Role            string            `json:"role" yaml:"role"`
	Status          string            `json:"status" yaml:"status"`
	Reason          string            `json:"reason" yaml:"reason"`
	PodName         string            `json:"podName" yaml:"podName"`
	Labels          map[string]string `json:"labels" yaml:"labels"`
	OwnerKind       string            `json:"ownerKind" yaml:"ownerKind"`
	OwnerApiVersion string            `json:"ownerApiVersion" yaml:"ownerApiVersion"`
	OwnerName       string            `json:"ownerName" yaml:"ownerName"`
	Updated         string            `json:"updated" yaml:"updated"`
	Sudo            bool              `json:"sudo" yaml:"sudo"`
	DaemonServerPid int               `json:"daemonserverpid" yaml:"daemonserverpid"`
	ServiceType     string            `json:"servicetype" yaml:"servicetype"`
}

func (s *SvcProfileV2) GetName() string {
	if s.Name == "" {
		if s.ActualName != "" {
			s.Name = s.ActualName
		}
	}
	return s.Name
}

func (s *SvcProfileV2) GetType() string {
	return s.Type
}
