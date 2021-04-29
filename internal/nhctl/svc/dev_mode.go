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
	"context"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func (c *Controller) GetDefaultPodName(ctx context.Context) (string, error) {
	var (
		podList *corev1.PodList
		err     error
	)
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", c.Name))
		default:
			switch c.Type { // todo hxx
			case appmeta.Deployment:
				podList, err = c.Client.ListPodsByDeployment(c.Name)
				if err != nil {
					return "", err
				}
			case appmeta.StatefulSet:
				podList, err = c.Client.ListPodsByStatefulSet(c.Name)
				if err != nil {
					return "", err
				}
			default:
				return "", errors.New(fmt.Sprintf("Service type %s not support", c.Type))
			}
		}
		if podList == nil || len(podList.Items) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", c.Name)
			time.Sleep(time.Second)
		} else {
			return podList.Items[0].Name, nil
		}
	}
}
