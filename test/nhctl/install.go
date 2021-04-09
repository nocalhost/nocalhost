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
