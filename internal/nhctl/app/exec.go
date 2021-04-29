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

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/log"
)

// Try to use shell defined in devContainerShell to enter pod's terminal
// If devContainerShell is not defined or shell defined in devContainerShell failed to enter terminal, use /bin/sh
// If container not specified, the first container will be used
func (a *Application) EnterPodTerminal(svcName string, podName, container string) error {
	pod := podName
	if pod == "" {
		podList, err := a.client.ListLatestRevisionPodsByDeployment(svcName)
		if err != nil {
			return err
		}
		if len(podList) != 1 {
			log.Warnf("The number of pods of %s is not 1 ???", svcName)
			return errors.New(fmt.Sprintf("The number of pods of %s is not 1 ???", svcName))
		}
		pod = podList[0].Name
	}
	shell := ""
	appProfile, _ := a.GetProfile()
	profile := appProfile.FetchSvcProfileV2FromProfile(svcName)
	if profile != nil {
		devConfig := profile.GetContainerDevConfigOrDefault(container)
		if devConfig != nil {
			shell = devConfig.Shell
		}
	}
	cmd := "(zsh || bash || sh)"
	if shell != "" {
		cmd = fmt.Sprintf("(%s || zsh || bash || sh)", shell)
	}
	return a.client.ExecShell(pod, container, cmd)
}

func (a *Application) Exec(svcName string, container string, commands []string) error {
	podList, err := a.client.ListPodsByDeployment(svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	return a.client.Exec(pod, container, commands)
}
