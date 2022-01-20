/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package controller

import "nocalhost/internal/nhctl/profile"

const (
//DevImageRevisionAnnotationKey            = "nhctl.dev.image.revision"
//DevImageOriginalPodReplicasAnnotationKey = "nhctl.dev.image.original.pod.replicas"
//DevImageRevisionAnnotationValue          = "first"
//
//PersistentVolumeDirLabel = "nocalhost.dev/dir"
//ServiceLabel             = "nocalhost.dev/service"
//AppLabel                 = "nocalhost.dev/app"
//
//DefaultSideCarImage = "nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"
//
//DefaultApplicationSyncPidFile = "syncthing.pid"
)

type ContainerDevEnv struct {
	DevEnv []*profile.Env
}
