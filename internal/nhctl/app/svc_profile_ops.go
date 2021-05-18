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
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/profile"
)

func (a *Application) GetSvcProfile(svcName string, svcType appmeta.SvcType) (*profile.SvcProfileV2, error) {
	profileV2, err := a.GetProfile()
	if err != nil {
		return nil, err
	}
	svcProfile := profileV2.SvcProfileV2(svcName, string(svcType))
	if svcProfile == nil {
		return nil, errors.New(
			fmt.Sprintf(
				"Svc Profile not found %s-%s-%s", profileV2.Namespace, svcType.String(), svcName,
			),
		)
	}
	return svcProfile, nil
}
