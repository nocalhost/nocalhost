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
	"errors"
	"nocalhost/test/nhctl"
	"nocalhost/test/util"
	"time"
)

func main() {
	commitId := nhctl.GetCommitId()
	if commitId == "" {
		panic(errors.New("this is should not happen"))
	}
	go util.TimeoutChecker(1 * time.Hour)
	nhctl.InstallNhctl(commitId)
	go nhctl.Init()
	if i := <-nhctl.StatusChan; i != 1 {
		nhctl.StopChan <- 1
	}
	defer nhctl.UninstallBookInfo()
	nhctl.InstallBookInfo()
	nhctl.PortForward()
	module := "details"
	nhctl.Dev(module)
	nhctl.Sync(module)
	nhctl.End(module)
}
