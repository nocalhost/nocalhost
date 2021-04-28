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

type DevStartOptions struct {
	WorkDir      string
	SideCarImage string
	DevImage     string
	Container    string
	//SvcType      string

	Kubeconfig string

	// for debug
	SyncthingVersion string

	// Now it's only use to specify the `root dir` user want to sync
	LocalSyncDir  []string
	StorageClass  string
	PriorityClass string
}

type FileSyncOptions struct {
	//RunAsDaemon    bool
	SyncDouble     bool
	SyncedPattern  []string
	IgnoredPattern []string
	Override       bool
	Container      string // container name of pod to sync
	Resume         bool
	Stop           bool
}

type SyncStatusOptions struct {
	Override bool
}
