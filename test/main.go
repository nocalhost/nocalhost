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

package main

import (
	"nocalhost/test/nhctlcli/suite"
)

func main() {
	cli, _, v2, cancelFunc := suite.Prepare()
	// ---------base line-----------
	t := suite.T{Cli: cli, CleanFunc: cancelFunc}
	t.Run("install", suite.Install)
	t.Run("dev", suite.Dev)
	t.Run("port-forward", suite.PortForward)
	t.Run("sync", suite.Sync)
	t.Run("upgrade", suite.Upgrade)
	t.Run("reset", suite.Reset)
	t.Run("apply", suite.Apply)
	t.Run("compatible", suite.Compatible, v2)
}
