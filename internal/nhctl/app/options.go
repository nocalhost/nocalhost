/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

type DevStartOptions struct {
	WorkDir      string
	SideCarImage string
	DevImage     string
	DevLang      string
	Namespace    string
	Kubeconfig   string
	LocalSyncDir []string
}

type FileSyncOptions struct {
	//RemoteDir         string
	//LocalSharedFolder string
	//LocalSshPort           int
	RemoteSyncthingPort    int
	RemoteSyncthingGUIPort int
	LocalSyncthingPort     int
	LocalSyncthingGUIPort  int
	RunAsDaemon            bool
	SyncDouble             bool
}
