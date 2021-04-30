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

package controller

import (
	"errors"
	"fmt"
)

// EnterPodTerminal Try to use shell defined in devContainerShell to enter pod's terminal
// If devContainerShell is not defined or shell defined in devContainerShell failed to enter terminal, use /bin/sh
// If container not specified, the first container will be used
func (c *Controller) EnterPodTerminal(podName, container string) error {
	pod := podName
	if pod == "" {
		// todo hxx

		podList, err := c.Client.ListLatestRevisionPodsByDeployment(c.Name)
		if err != nil {
			return err
		}
		if len(podList) != 1 {
			return errors.New(fmt.Sprintf("The number of pods of %s is not 1 ???", c.Name))
		}
		pod = podList[0].Name

	}
	shell := ""
	profile, _ := c.GetProfile()
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
	return c.Client.ExecShell(pod, container, cmd)
}
