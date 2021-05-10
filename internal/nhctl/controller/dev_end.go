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
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func (c *Controller) DevEnd(reset bool) error {
	if err := c.BuildPodController().RollBack(reset); err != nil {
		if !reset {
			return err
		}
		log.WarnE(err, "something incorrect occurs when rolling back")
	}

	utils.ShouldI(c.AppMeta.SvcDevEnd(c.Name, c.Type), "something incorrect occurs when updating secret")
	utils.ShouldI(c.StopSyncAndPortForwardProcess(true), "something incorrect occurs when stopping sync process")
	return nil
}
