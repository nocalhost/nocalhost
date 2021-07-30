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
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/clientgoutils"
)

func StandardNocalhostMetas(releaseName, releaseNamespace string) *clientgoutils.ApplyFlags {
	return &clientgoutils.ApplyFlags{
		MergeableLabel: map[string]string{
			_const.AppManagedByLabel: _const.AppManagedByNocalhost,
		},

		MergeableAnnotation: map[string]string{
			_const.NocalhostApplicationName:      releaseName,
			_const.NocalhostApplicationNamespace: releaseNamespace,
		},
		DoApply: true,
	}
}
