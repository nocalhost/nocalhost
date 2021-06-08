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

package nocalhost

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

	DefaultSideCarImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"

	DefaultApplicationSyncPidFile = "syncthing.pid"

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application " +
		"to managed that those manifest out of Nocalhost" +
		" management so can't be install, uninstall, reset, etc."

	HelmReleaseName = "meta.helm.sh/release-name"

	DevWorkloadIgnored = "nocalhost.dev.workload.ignored"
)
