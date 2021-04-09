package nhctl

import (
	v1 "k8s.io/api/core/v1"
	"nocalhost/test/util"
	"testing"
)

func TestInstallBookInfo(t *testing.T) {
	cmd := "nhctl uninstall bookinfo -n test --force --kubeconfig " + util.CODING
	util.WaitForCommandDone(cmd)
	installBookInfoHelmGit()
	installBookInfoKustomizeGit()
	installBookInfoRawManifest()
	PortForward()
}

func TestWait(t *testing.T) {
	util.WaitToBeStatus(
		"test",
		"pods",
		"app=details",
		func(i interface{}) bool {
			return i.(*v1.Pod).Status.Phase == v1.PodRunning && func() bool {
				for _, containerStatus := range i.(*v1.Pod).Status.ContainerStatuses {
					if containerStatus.Ready {
						return false
					}
				}
				return true
			}()
		})
}
