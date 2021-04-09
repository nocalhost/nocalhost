package nhctl

import (
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"nocalhost/test/util"
	"os"
	"strings"
	"time"
)

func Dev(moduleName string) {
	if err := os.MkdirAll(fmt.Sprintf("/tmp/%s", moduleName), 0777); err != nil {
		panic(fmt.Sprintf("test case failed, reason: create directory error, error: %v\n", err))
	}
	cmd := "nhctl dev start bookinfo -d %s -s /tmp/%s --priority-class nocalhost-container-critical -n test --kubeconfig " + util.CODING
	if ok, _ := util.WaitForCommandDone(fmt.Sprintf(cmd, moduleName, moduleName)); !ok {
		panic("test case failed, reason: nhctl dev start failed, command: " + cmd)
	}
	util.WaitToBeStatus("test", "pods", "app="+moduleName, func(i interface{}) bool {
		return i.(*v1.Pod).Status.Phase == v1.PodRunning
	})
}

func Sync(moduleName string) {
	cmd := fmt.Sprintf("nhctl sync bookinfo -d %s -n test --kubeconfig="+util.CODING, moduleName)
	ok, log := util.WaitForCommandDone(cmd)
	if !ok {
		panic(fmt.Sprintf("sync failed, reason: sync file failed, command: %s, log: %s\n", cmd, log))
	}

	filename := "hello.test"
	content := "this is a test"
	if err := ioutil.WriteFile(fmt.Sprintf("/tmp/%s/%s", moduleName, filename), []byte(content), 0644); err != nil {
		panic(fmt.Sprintf("test case failed, reason: write file %s error: %v\n", filename, err))
	}
	// wait file to be synchronize
	time.Sleep(10 * time.Second)
	cmd = fmt.Sprintf("kubectl exec deployment/%s -n test --kubeconfig=%s -- cat %s\n", moduleName, util.CODING, filename)
	ok, log = util.WaitForCommandDone(cmd)
	if !ok {
		panic(fmt.Sprintf("test case failed, reason: cat file %s error, command: %s, log: %v\n", filename, cmd, log))
	}
	if !strings.Contains(log, content) {
		panic(fmt.Sprintf("test case failed, reason: file content: %s not equals command log: %s\n", content, log))
	}
}

func PortForward() {
	retry := 100
	req, _ := http.NewRequest("GET", "http://localhost:39080", nil)
	for i := 0; i < retry; i++ {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err)
		}
		if res == nil || res.StatusCode != 200 {
			time.Sleep(2 * time.Second)
		} else {
			return
		}
	}
	panic(fmt.Sprintf("test case failed, reason: can't open productpage"))
}

func End(moduleName string) {
	cmd := "nhctl dev end bookinfo -d %s -n test --kubeconfig " + util.CODING
	if ok, log := util.WaitForCommandDone(fmt.Sprintf(cmd, moduleName)); !ok {
		panic(fmt.Sprintf("test case failed, reason: nhctl dev end failed, command: %s, log: %s \n", cmd, log))
	}
	util.WaitToBeStatus("test", "pods", "app=details", func(i interface{}) bool {
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
