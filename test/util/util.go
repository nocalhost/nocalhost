package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	clientgowatch "k8s.io/client-go/tools/watch"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os/exec"
	"strings"
	"time"
)

var Client *clientgoutils.ClientGoUtils

const CODING = "/root/.kube/config"

//const CODING = "/Users/naison/codingtest"
//const CODING = "/Users/naison/.kube/config"

func init() {
	temp, err := clientgoutils.NewClientGoUtils(CODING, "test")
	if err != nil {
		panic(fmt.Sprintf("init k8s client error: %v\n", err))
	}
	Client = temp
}

func WaitForCommandDone(command string) (bool, string) {
	fmt.Println(command)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var b strings.Builder
	stdoutRead, err := cmd.StdoutPipe()
	if err != nil {
		return false, err.Error()
	}
	if err = cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		return false, err.Error()
	}
	defer cmd.Wait()
	defer stdoutRead.Close()
	lineBody := bufio.NewReaderSize(stdoutRead, 1024)
	go func() {
		for {
			line, isPrefix, err := lineBody.ReadLine()
			if err != nil && err == io.EOF {
				//fmt.Printf("error: %v", err)
				break
			}
			if len(line) != 0 && !isPrefix {
				fmt.Println(string(line))
				b.WriteString(string(line))
			}
		}
	}()
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Println("Command timeout")
		return false, b.String()
	}
	_ = cmd.Wait()
	time.Sleep(2 * time.Second)
	return cmd.ProcessState.Success(), b.String()
}

func WaitToBeStatus(namespace string, resource string, label string, checker func(interface{}) bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	watchlist := cache.NewFilteredListWatchFromClient(
		Client.ClientSet.CoreV1().RESTClient(),
		resource,
		namespace,
		func(options *metav1.ListOptions) {
			options.LabelSelector = label
		})

	preConditionFunc := func(store cache.Store) (bool, error) {
		if len(store.List()) == 0 {
			return false, nil
		}
		for _, p := range store.List() {
			if !checker(p) {
				return false, nil
			}
		}
		return true, nil
	}

	conditionFunc := func(e watch.Event) (bool, error) {
		return checker(e.Object), nil
	}
	event, err := clientgowatch.UntilWithSync(ctx, watchlist, &v1.Pod{}, preConditionFunc, conditionFunc)
	if err != nil {
		fmt.Printf("wait to ready failed, error: %v, event: %v\n", err, event)
		return false
	}
	return true
}
