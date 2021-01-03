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
	"context"
	"fmt"

	"github.com/pkg/errors"

	"nocalhost/pkg/nhctl/log"
)

func (a *Application) EnterPodTerminal(svcName string) error {
	podList, err := a.client.ListPodsOfLatestRevisionByDeployment(a.GetNamespace(), svcName)
	//podList, err := a.client.GetPodsFromDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		return err
	}
	if len(podList) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList[0].Name
	return a.client.ExecBash(a.GetNamespace(), pod, "")
}

func (a *Application) Exec(svcName string, commands []string) error {
	podList, err := a.client.GetPodsFromDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	return a.client.Exec(a.GetNamespace(), pod, "", commands)
}
