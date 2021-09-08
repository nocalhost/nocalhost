/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import "time"

const (
	DefaultSecretGenSign               = "secreted"
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
	DefaultInitHelmCODINGGitRepo       = "https://e.coding.net/nocalhost/nocalhost/nocalhost.git"
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
	DefaultInitApplicationHelm = "{\"source\":\"git\",\"install_type\":\"helm_chart\"," +
		"\"application_url\":\"git@github.com:nocalhost/bookinfo.git\"," +
		"\"application_config_path\":\"config.helm.yaml\",\"application_name\":" +
		"\"bookinfo-helm\",\"resource_dir\":[]}"
	DefaultInitApplicationKustomize = "{\"source\":\"git\",\"install_type\":\"kustomize\"," +
		"\"application_name\":\"bookinfo-kustomize\",\"application_url\":" +
		"\"git@github.com:nocalhost/bookinfo.git\",\"application_config_path\"" +
		":\"config.kustomize.yaml\",\"resource_dir\":[]}"
	DefaultInitApplicationCODING = "{\"source\":\"git\",\"install_type\":" +
		"\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\"" +
		":\"bookinfo\",\"application_url\":" +
		"\"https://e.coding.net/nocalhost/nocalhost/bookinfo.git\"}"
	DefaultInitApplicationBookinfoTracing = "{\"source\":\"git\",\"install_type\":\"rawManifest\"," +
		"\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo-tracing\"," +
		"\"application_url\":\"https://github.com/nocalhost/bookinfo-tracing.git\"}"
	// Init Component Version Control, HEAD means build from tag
	DefaultNocalhostMainBranch        = "HEAD"
	DefaultNocalhostDepDockerRegistry = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-dep"

	DefaultDevContainerShell = "(zsh || bash || sh)"

	DependenceConfigMapPrefix = "nocalhost-depends-do-not-overwrite"

	// Port-forward
	PortForwardManual   = "manual"
	PortForwardDevPorts = "devPorts"
)
