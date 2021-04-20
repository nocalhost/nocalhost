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

package nhctl

import (
	"fmt"
	"nocalhost/test/util"
	"testing"
	"time"
)

func TestInstallNhctl(t *testing.T) {
	InstallNhctl("1f820388196a2bc57a7d118d46c40e9f99c8c119")
}

func TestInit(t *testing.T) {
	go Init()
	for {
		select {
		case status := <-StatusChan:
			if status == 0 {
				fmt.Println("ok")
				//StopChan <- 0
			} else {
				fmt.Printf("not ok")
			}
		}
	}
}

func TestCommandWait(t *testing.T) {
	go util.WaitForCommandDone("kubectl wait --for=condition=Delete pods/test-74cb446f4b-hmp8g -n test")
	time.Sleep(7 * time.Second)
}

func TestDev(t *testing.T) {
	moduleName := "details"
	Dev(moduleName)
	Sync(moduleName)
	End(moduleName)
}
