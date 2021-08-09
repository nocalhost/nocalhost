/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package _const

const (
	NocalhostApplicationName      = "dev.nocalhost/application-name"
	NocalhostApplicationNamespace = "dev.nocalhost/application-namespace"
	AppManagedByLabel             = "app.kubernetes.io/managed-by"
	AppManagedByNocalhost         = "nocalhost"
	DefaultNocalhostSideCarName   = "nocalhost-sidecar"
	DefaultSidecarImagePullPolicy = "Always"

	DevImageRevisionAnnotationKey            = "nhctl.dev.image.revision"
	DevImageOriginalPodReplicasAnnotationKey = "nhctl.dev.image.original.pod.replicas"
	DevImageRevisionAnnotationValue          = "first"

	PersistentVolumeDirLabel = "nocalhost.dev/dir"
	ServiceLabel             = "nocalhost.dev/service"
	AppLabel                 = "nocalhost.dev/app"

	DefaultSideCarImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:sshversion"

	DefaultApplicationSyncPidFile = "syncthing.pid"

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application " +
		"to managed that those manifest out of Nocalhost" +
		" management so can't be install, uninstall, reset, etc."

	HelmReleaseName = "meta.helm.sh/release-name"

	DevWorkloadIgnored = "nocalhost.dev.workload.ignored"

	NocalhostRoleBindingLabelKey = "owner"
	NocalhostRoleBindingLabelVal = "nocalhost"

	NocalhostDefaultSaNs        = "default"
	NocalhostDefaultRoleBinding = "nocalhost-role-binding"
	NocalhostDevRoleName        = "nocalhost-dev-role"

	NocalhostCooperatorRoleBinding = "nocalhost-cooperator-role-binding"
	NocalhostCooperatorRoleName    = "nocalhost-cooperator-role"

	NocalhostViewerRoleBinding = "nocalhost-viewer-role-binding"
	NocalhostViewerRoleName    = "nocalhost-viewer-role"
)
