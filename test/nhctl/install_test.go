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

package nhctl

import (
	v1 "k8s.io/api/core/v1"
	"nocalhost/test/util"
	"testing"
)

func TestInstallBookInfo(t *testing.T) {
	cmd := "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
	installBookInfoHelmGit()
	installBookInfoKustomizeGit()
	installBookInfoRawManifest()
	PortForward()
}

func TestWait(t *testing.T) {
	util.WaitToBeStatus(
		"test",
		"pods",
		"app=details",
		func(i interface{}) bool {
			return i.(*v1.Pod).Status.Phase == v1.PodRunning && func() bool {
				for _, containerStatus := range i.(*v1.Pod).Status.ContainerStatuses {
					if containerStatus.Ready {
						return false
					}
				}
				return true
			}()
		})
}
