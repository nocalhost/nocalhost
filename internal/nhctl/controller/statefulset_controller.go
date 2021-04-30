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
	"context"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
)

type StatefulSetController struct {
	*Controller
}

func (s *StatefulSetController) Name() string {
	return s.Controller.Name
}

func (s *StatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	panic("implement me")
}

func (s *StatefulSetController) ScaleReplicasToOne(ctx context.Context) error {
	scale, err := s.Client.GetStatefulSetClient().GetScale(ctx, s.Name(), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 {
		scale.Spec.Replicas = 1
		_, err = s.Client.GetStatefulSetClient().UpdateScale(ctx, s.Name(), scale, metav1.UpdateOptions{})
		return errors.Wrap(err, "")
	}
	return nil
}

// Container Get specify container
// If containerName not specified:
// 	 if there is only one container defined in spec, return it
//	 if there are more than one container defined in spec, return err
func (s *StatefulSetController) Container(containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	ss, err := s.Client.GetStatefulSet(s.Name())
	if err != nil {
		return nil, err
	}
	if containerName != "" {
		for index, c := range ss.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				return &ss.Spec.Template.Spec.Containers[index], nil
			}
		}
		if devContainer == nil {
			return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
		}
	} else {
		if len(ss.Spec.Template.Spec.Containers) > 1 {
			return nil, errors.New(fmt.Sprintf("There are more than one container defined," +
				"please specify one to start developing"))
		}
		if len(ss.Spec.Template.Spec.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &ss.Spec.Template.Spec.Containers[0]
	}
	return devContainer, nil
}

func (s *StatefulSetController) RollBack(reset bool) error {
	panic("implement me")
}

func (s *StatefulSetController) GetDefaultPodNameWait(ctx context.Context) (string, error) {
	return getDefaultPodName(ctx, s)
}

func (s *StatefulSetController) GetPodList() ([]corev1.Pod, error) {
	list, err := s.Client.ListPodsByStatefulSet(s.Name())
	if err != nil {
		return nil, err
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("no pod found")
	}
	return list.Items, nil
}
