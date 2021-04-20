/*
Copyright 2021 The Nocalhost Authors.
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

package testcase

import (
	v1 "k8s.io/api/core/v1"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/util"
	"testing"
)

func TestInstallBookInfo(t *testing.T) {
	cli := nhctlcli.NewNhctl("/Users/naison/codingtest", "test")
	cmd := "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
	installBookInfoHelmGit(cli)
	installBookInfoKustomizeGit(cli)
	installBookInfoRawManifest(cli)
	PortForwardCheck(1)
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

func TestJson(t *testing.T) {
	cli := nhctlcli.NewNhctl("/Users/naison/codingtest", "test")
	StatusCheck(cli, "details")
}
