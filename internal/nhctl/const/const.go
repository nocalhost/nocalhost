/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package _const

import "github.com/fatih/color"

const (
	UintEnable  = uint64(1)
	UintDisable = uint64(0)

	DefaultEmailSuffix = "@nocalhost.dev"

	DefaultNewFilePermission   = 0755
	DefaultBinDirName          = "bin"
	DefaultBinSyncThingDirName = "syncthing"
	DefaultLogDirName          = "logs"
	DefaultLogFileName         = "nhctl.log"

	NocalhostApplicationName         = "dev.nocalhost/application-name"
	NocalhostApplicationNamespace    = "dev.nocalhost/application-namespace"
	NocalhostDevContainerAnnotations = "dev.nocalhost/nocalhost-dev"

	NocalhostDefaultDevContainerName = "nocalhost-dev"
	NocalhostDefaultDevSidecarName   = "nocalhost-sidecar"

	OriginWorkloadDefinition    = "dev.nocalhost/origin-workload-definition"
	OriginProbeDefinition       = "dev.nocalhost/origin-probe-definition"
	DevModeCount                = "dev.nocalhost/dev-mode-count"
	AppManagedByLabel           = "app.kubernetes.io/managed-by"
	AppManagedByNocalhost       = "nocalhost"
	DefaultNocalhostSideCarName = "nocalhost-sidecar"

	PersistentVolumeDirLabel = "nocalhost.dev/dir"
	ServiceLabel             = "nocalhost.dev/service"
	ServiceTypeLabel         = "nocalhost.dev/service-type"
	AppLabel                 = "nocalhost.dev/app"

	DefaultSideCarImage = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"
	SSHSideCarImage     = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:sshversion"
	DefaultVPNImage     = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-vpn:v1"

	DefaultApplicationSyncPidFile = "syncthing.pid"

	EnableFullLogEnvKey = "NH_FULL_LOG"

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application " +
		"to managed that those manifest out of Nocalhost" +
		" management so can't be install, uninstall, reset, etc."

	HelmReleaseName = "meta.helm.sh/release-name"

	DevWorkloadIgnored = "nocalhost.dev.workload.ignored"

	NocalhostCmLabelKey   = "dep-management"
	NocalhostCmLabelValue = "nocalhost"

	NocalhostRoleBindingLabelKey = "owner"
	NocalhostRoleBindingLabelVal = "nocalhost"

	NocalhostDefaultSaNs        = "default"
	NocalhostDefaultRoleBinding = "nocalhost-role-binding"
	NocalhostDevRoleName        = "nocalhost-dev-role"

	NocalhostCooperatorRoleBinding = "nocalhost-cooperator-role-binding"
	NocalhostCooperatorRoleName    = "nocalhost-cooperator-role"

	NocalhostViewerRoleBinding = "nocalhost-viewer-role-binding"
	NocalhostViewerRoleName    = "nocalhost-viewer-role"

	HPAOriginalMaxReplicasKey = "nocalhost.dev.hpa.origin.max.replicas"
	HPAOriginalMinReplicasKey = "nocalhost.dev.hpa.origin.min.replicas"

	// sycnthing

	// sync type
	DefaultSyncType       = "sendReceive" // default sync mode
	SendOnlySyncType      = "sendonly"
	SendOnlySyncTypeAlias = "send"

	// sync mode
	GitIgnoreMode = "gitIgnore"
	PatternMode   = "pattern"

	banner = `
****************************************
*      Nocalhost DevMode Terminal      *
****************************************
`
)

func BoolToUint64Pointer(bool bool) *uint64 {
	var result uint64
	if bool {
		result = UintEnable
	} else {
		result = UintDisable
	}
	return &result
}

func Uint64PointerToBool(value *uint64) bool {
	return value != nil && *value != UintDisable
}

var (
	IsDaemon              = false
	DevModeTerminalBanner = color.New(color.BgGreen).Add(color.FgBlack).Add(color.Bold).Sprint(banner)
)
