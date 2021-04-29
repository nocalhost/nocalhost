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

	DevImageRevisionAnnotationKey            = "nhctl.dev.image.revision"
	DevImageOriginalPodReplicasAnnotationKey = "nhctl.dev.image.original.pod.replicas"
	DevImageRevisionAnnotationValue          = "first"

	PersistentVolumeDirLabel = "nocalhost.dev/dir"
	ServiceLabel             = "nocalhost.dev/service"
	AppLabel                 = "nocalhost.dev/app"

	DefaultSideCarImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"

	DefaultApplicationSyncPidFile = "syncthing.pid"
)
