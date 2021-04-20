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

package app

import "time"

const (
	DefaultSideCarImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"
	//DefaultDevImage            = "codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	//DefaultWorkDir             = "/home/nocalhost-dev"
	DefaultUpgradeResourcesDir = "upgrade_resources"
	//DefaultNhctlHomeDirName                  = ".nh/nhctl"
	//DefaultBinDirName                        = "bin"
	//DefaultLogDirName                        = "logs"
	DefaultSyncLogFileName                   = "sync-port-forward-child-process.log"
	DefaultApplicationSyncPortForwardPidFile = "sync-port-forward.pid"
	//DefaultBinSyncThingDirName               = "syncthing"
	DefaultBackGroundPortForwardLogFileName  = "alone-port-forward-child-process.log"
	DefaultApplicationOnlyPortForwardPidFile = "alone-port-forward.pid"
	GetFileLockPath                          = "lock"
	DefaultApplicationSyncPidFile            = "syncthing.pid"
	//DefaultApplicationDirName                = "application"

	DefaultApplicationConfigPath       = ".config.yaml"
	DefaultApplicationConfigV2Path     = ".config_v2.yaml"
	DefaultGitNocalhostDir             = ".nocalhost" // DefaultApplicationConfigDirName
	DefaultConfigNameInGitNocalhostDir = "config.yaml"
	DefaultNewFilePermission           = 0700
	DefaultConfigFilePermission        = 0644
	DefaultClientGoTimeOut             = time.Minute * 5

	// nhctl init
	// TODO when release
	DefaultInitHelmGitRepo             = "https://github.com/nocalhost/nocalhost.git"
	DefaultInitHelmCODINGGitRepo       = "https://e.coding.net/codingcorp/nocalhost/nocalhost.git"
	DefaultInitHelmType                = "helmGit"
	DefaultInitWatchDeployment         = "nocalhost-api"
	DefaultInitWatchWebDeployment      = "nocalhost-web"
	DefaultInitNocalhostService        = "nocalhost-web"
	DefaultInitInstallApplicationName  = "nocalhost"
	DefaultInitUserEmail               = "foo@nocalhost.dev"
	DefaultInitMiniKubePortForwardPort = 31219
	DefaultInitPassword                = "123456"
	DefaultInitAdminUserName           = "admin@admin.com"
	DefaultInitAdminPassWord           = "123456"
	DefaultInitName                    = "Nocalhost"
	DefaultInitWaitNameSpace           = "nocalhost-reserved"
	DefaultInitCreateNameSpaceLabels   = "nocalhost-init"
	DefaultInitWaitDeployment          = "nocalhost-dep"
	// TODO when release
	DefaultInitHelmResourcePath   = "deployments/chart"
	DefaultInitPortForwardTimeOut = time.Minute * 1
	DefaultInitApplicationGithub  = "{\"source\":\"git\",\"install_type\":\"rawManifest\"," +
		"\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo\"," +
		"\"application_url\":\"https://github.com/nocalhost/bookinfo.git\"}"
	DefaultInitApplicationHelm      = "{\"source\":\"git\",\"install_type\":\"helm_chart\"," +
		"\"application_url\":\"git@github.com:nocalhost/bookinfo.git\"," +
		"\"application_config_path\":\"config.helm.yaml\",\"application_name\":" +
		"\"bookinfo-helm\",\"resource_dir\":[]}"
	DefaultInitApplicationKustomize = "{\"source\":\"git\",\"install_type\":\"kustomize\"," +
		"\"application_name\":\"bookinfo-kustomize\",\"application_url\":" +
		"\"git@github.com:nocalhost/bookinfo.git\",\"application_config_path\"" +
		":\"config.kustomize.yaml\",\"resource_dir\":[]}"
	DefaultInitApplicationCODING    = "{\"source\":\"git\",\"install_type\":" +
		"\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\"" +
		":\"bookinfo\",\"application_url\":" +
		"\"https://e.coding.net/codingcorp/nocalhost/bookinfo.git\"}"
	// Init Component Version Control, HEAD means build from tag
	DefaultNocalhostMainBranch        = "HEAD"
	DefaultNocalhostDepDockerRegistry = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-dep"

	// file sync
	DefaultNocalhostSideCarName = "nocalhost-sidecar"

	DefaultDevContainerShell = "(zsh || bash || sh)"

	DependenceConfigMapPrefix = "nocalhost-depends-do-not-overwrite"

	// Port-forward
	PortForwardManual   = "manual"
	PortForwardDevPorts = "devPorts"
)
