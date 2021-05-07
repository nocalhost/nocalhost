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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
)

// Controller presents a k8s controller
// https://kubernetes.io/docs/concepts/architecture/controller
type Controller struct {
	NameSpace string
	AppName   string
	Name      string
	Type      appmeta.SvcType
	Client    *clientgoutils.ClientGoUtils
	AppMeta   *appmeta.ApplicationMeta
}

func (c *Controller) IsInDevMode() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Type)
}

func (c *Controller) CheckIfExist() (bool, error) {
	var err error
	switch c.Type {
	case appmeta.Deployment:
		_, err = c.Client.GetDeployment(c.Name)
	case appmeta.StatefulSet:
		_, err = c.Client.GetStatefulSet(c.Name)
	case appmeta.DaemonSet:
		_, err = c.Client.GetDaemonSet(c.Name)
	case appmeta.Job:
		_, err = c.Client.GetJobs(c.Name)
	case appmeta.CronJob:
		_, err = c.Client.GetCronJobs(c.Name)
	default:
		return false, errors.New("unsupported controller type")
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *Controller) GetDescription() *profile.SvcProfileV2 {
	//appProfile, err := c.GetAppProfile()
	//if err != nil {
	//	return ""
	//}
	//svcProfile :=Ø appProfile.SvcProfileV2(c.Name, string(c.Type))
	//desc := ""
	//if svcProfile != nil {
	//	svcProfile.Developing = c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Type)
	//	svcProfile.Possess = c.AppMeta.SvcDevModePossessor(c.Name, c.Type, appProfile.Identifier)
	//	bytes, err := yaml.Marshal(svcProfile)
	//	if err == nil {
	//		desc = string(bytes)
	//	}
	//}
	//return desc

	appProfile, err := c.GetAppProfile()
	if err != nil {
		return nil
	}
	svcProfile := appProfile.SvcProfileV2(c.Name, string(c.Type))
	if svcProfile != nil {
		svcProfile.Developing = c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Type)
		svcProfile.Possess = c.AppMeta.SvcDevModePossessor(c.Name, c.Type, appProfile.Identifier)
		return svcProfile
	}
	return nil
}
