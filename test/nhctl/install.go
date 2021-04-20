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
)

func InstallBookInfo() {
	cmd := "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
	//installBookInfoHelmGit()
	installBookInfoKustomizeGit()
	installBookInfoRawManifest()
}

func UninstallBookInfo() {
	cmd := "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
}

func installBookInfoRawManifest() {
	cmd := "nhctl install bookinfo -u https://github.com/nocalhost/bookinfo.git -t rawManifest -n test --resource-path manifest/templates --kubeconfig " + util.CODING
	if ok, _ := util.WaitForCommandDone(cmd); !ok {
		panic(fmt.Sprintf("\ntest case failed, reason: nhctl install bookinfo error, command: %s\n", cmd))
	}
}

func installBookInfoHelmGit() {
	cmd := "nhctl install bookinfo -u https://github.com/nocalhost/bookinfo.git -t helmGit  -n test  --resource-path charts/bookinfo --kubeconfig=" + util.CODING
	if ok, _ := util.WaitForCommandDone(cmd); !ok {
		panic(fmt.Sprintf("\ntest case failed, reason: nhctl install helmGit bookinfo error, command: %s\n", cmd))
	}
	cmd = "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
}

func installBookInfoKustomizeGit() {
	cmd := "nhctl install bookinfo -u https://github.com/nocalhost/bookinfo.git -t kustomizeGit -n test  --resource-path kustomize/base --kubeconfig=" + util.CODING
	if ok, _ := util.WaitForCommandDone(cmd); !ok {
		panic(fmt.Sprintf("\ntest case failed, reason: nhctl install kustomizeGit bookinfo error, command: %s\n", cmd))
	}
	cmd = "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
}
