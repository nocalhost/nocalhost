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

import (
	"fmt"

	"github.com/pkg/errors"

	"nocalhost/pkg/nhctl/log"
)

// Try to use shell defined in devContainerShell to enter pod's terminal
// If devContainerShell is not defined or shell defined in devContainerShell failed to enter terminal, use /bin/sh
func (a *Application) EnterPodTerminal(svcName string) error {
	podList, err := a.client.ListPodsOfLatestRevisionByDeployment(svcName)
	if err != nil {
		return err
	}
	if len(podList) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList[0].Name
	shell := a.GetSvcProfile(svcName).DevContainerShell
	if shell != "" {
		log.Debugf("Shell %s defined, use it to enter terminal", shell)
		err = a.client.ExecShell(pod, "", shell)
		if err != nil {
			log.Warnf("Failed to use %s to enter terminal, use %s instead", shell, DefaultDevContainerShell)
		} else {
			return nil
		}
	}
	if shell == "" {
		log.Debugf("Shell not defined, use default shell %s to enter terminal", shell)
	}
	return a.client.ExecShell(pod, "", DefaultDevContainerShell)
}

func (a *Application) Exec(svcName string, commands []string) error {
	podList, err := a.client.GetPodsFromDeployment(svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	return a.client.Exec(pod, "", commands)
}
