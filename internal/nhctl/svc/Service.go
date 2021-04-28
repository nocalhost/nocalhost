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

package svc

import (
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/clientgoutils"
)

type Service struct {
	NameSpace string
	AppName   string
	Name      string
	Type      appmeta.SvcType
	Client    *clientgoutils.ClientGoUtils
	AppMeta   *appmeta.ApplicationMeta
}

func (s *Service) IsInDevMode() bool {
	return s.AppMeta.CheckIfSvcDeveloping(s.Name, s.Type)
}

func (s *Service) CheckIfExist() (bool, error) {
	var err error
	switch s.Type {
	case appmeta.Deployment:
		_, err = s.Client.GetDeployment(s.Name)
	case appmeta.StatefulSet:
		_, err = s.Client.GetStatefulSet(s.Name)
	case appmeta.DaemonSet:
		_, err = s.Client.GetDaemonSet(s.Name)
	case appmeta.Job:
		_, err = s.Client.GetJobs(s.Name)
	case appmeta.CronJob:
		_, err = s.Client.GetCronJobs(s.Name)
	default:
		return false, errors.New("unsupported svc type")
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}
