package nhctl

import (
	"bufio"
	"context"
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"strings"
	"time"
)

var StopChan = make(chan int32, 1)
var StatusChan = make(chan int32, 1)

// pod's create timestamp is priority, first create should first run. needs to judge first pod is this pod or not
// if first pod is this pod, then this pod run, else wait for
func GetCommitId() string {
	// this value will be set in github action workflow
	id := os.Getenv("COMMIT_ID")
	fmt.Printf("get id from env: %v\n", id)
	for {
		ids := getAllCommitId()
		if len(ids) <= 1 {
			return id
		}
		var minTimestamp = time.Now().Add(1 * time.Hour).UnixNano()
		exist := false
		for timestamp, commitId := range ids {
			if commitId == id {
				exist = true
			}
			if minTimestamp > timestamp {
				minTimestamp = timestamp
			}
		}
		if !exist {
			panic("this should not happen")
		}
		if ids[minTimestamp] == id {
			return id
		}
		time.Sleep(10 * time.Second)
	}
}

// get all pods, pod name is taskId, we will find the first create pod, judge it's my turns ro not
func getAllCommitId() map[int64]string {
	podList, err := util.Client.ClientSet.CoreV1().Pods("test").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=test",
	})
	if err != nil {
		panic(fmt.Sprintf("get commit configmap error: %v\n", err))
	}
	var order = make(map[int64]string)
	for _, pod := range podList.Items {
		priority := pod.CreationTimestamp.UnixNano()
		order[priority] = pod.Name
	}
	return order
}

func InstallNhctl(commitId string) {
	str := "curl --fail -L \"https://codingcorp-generic.pkg.coding.net/nocalhost/nhctl/nhctl-linux-amd64?version=%s\" -o nhctl"
	fmt.Printf(str, commitId)
	cmd := exec.Command("sh", "-c", fmt.Sprintf(str, commitId))
	var output []byte
	var err error
	if output, err = cmd.Output(); err != nil {
		_ = cmd.Process.Kill()
		panic(fmt.Sprintf("download nhctl error: %v\n", err))
	}
	fmt.Printf("download nhctl output: %v\n", string(output))

	cmd = exec.Command("sh", "-c", "chmod +x nhctl")
	if output, err = cmd.Output(); err != nil {
		_ = cmd.Process.Kill()
		panic(fmt.Sprintf("chmod nhctl error: %v\n", err))
	}

	cmd = exec.Command("sh", "-c", "mv ./nhctl /usr/local/bin/nhctl")
	if output, err = cmd.Output(); err != nil {
		_ = cmd.Process.Kill()
		panic(fmt.Sprintf("mv nhctl error: %v\n", err))
	}
}

func Init() {
	var c = "nhctl init demo -n nocalhost -p 7000 --force --kubeconfig=" + util.CODING
	cmd := exec.Command("sh", "-c", c)
	stdoutRead, err := cmd.StdoutPipe()
	if err != nil {
		panic(errors.Wrap(err, "stdout error"))
	}
	if err := cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		panic(fmt.Sprintf("nhctl init error: %v\n", err))
	}
	defer cmd.Wait()
	defer stdoutRead.Close()
	lineBody := bufio.NewReaderSize(stdoutRead, 1024)
	go func() {
		for {
			line, isPrefix, err := lineBody.ReadLine()
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			if len(line) != 0 && !isPrefix {
				fmt.Println(string(line))
			}
			if strings.Contains(string(line), "Nocalhost init completed") {
				StatusChan <- 1
				break
			}
		}
	}()
	for {
		select {
		case stat := <-StopChan:
			switch stat {
			case 0: // ok
				_ = cmd.Process.Kill()
				return
			default:
				_ = cmd.Process.Kill()
				panic("exit not ok")
			}
		}
	}
}
