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
	var name string
	for _, c := range podList.Items[0].Spec.Containers {
		if c.Name == "nocalhost-dev" {
			name = c.Name
		}
		if c.Name == container {
			name = container
			break
		}
	}
	if len(name) == 0 {
		return errors.New(fmt.Sprintf("container: %s not found", name))
	}
	return a.client.Exec(pod, name, commands)
}
