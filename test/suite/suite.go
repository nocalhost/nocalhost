/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package suite

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"net/http"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/testcase"
	"nocalhost/test/util"
	"os"
	"strings"
	"time"
)

// test suite
type T struct {
	Cli       runner.Client
	CleanFunc func()
}

func NewT(namespace, kubeconfig string, f func()) *T {
	return &T{
		Cli:       runner.NewClient(kubeconfig, namespace, "Main"),
		CleanFunc: f,
	}
}

func (t *T) Run(name string, fn func(cli runner.Client)) {
	t.RunWithBookInfo(true, name, fn)
}

// Run command and clean environment after finished
func (t *T) RunWithBookInfo(withBookInfo bool, name string, fn func(cli runner.Client)) {
	logger := log.TestLogger(name)

	logger.Infof("============= Testing (Start)%s  =============\n", name)
	timeBefore := time.Now()

	defer func() {
		if err := recover(); err != nil {
			t.AlertForImagePull()
			LogsForArchive()
			logger.Infof(">>> Panic on suit %s <<<", name)

			t.Clean()
			t.Alert()
			panic(err)
		}
	}()

	clientForRunner := t.Cli.RandomNsCli(name)
	if err := util.RetryFunc(
		func() error {
			result, errOutput, err := clientForRunner.GetKubectl().RunClusterScope(
				context.Background(), "create", "ns", clientForRunner.NameSpace(),
			)

			if k8serrors.IsAlreadyExists(err) || strings.Contains(errOutput, "already exists") {
				return nil
			}

			if strings.Contains(result, "created") {
				return nil
			}

			return errors.Wrap(err, "Error while create ns: "+errOutput)
		},
	); err != nil {
		fmt.Printf("Error stack: %v\n", err)
		panic(err)
		return
	}

	logger.Infof("============= Testing (Create Ns)%s  =============\n", name)

	var retryTimes = 5
	if withBookInfo {
		defer func() {
			_ = testcase.UninstallBookInfo(clientForRunner)
		}()

		var err error
		for i := 0; i < retryTimes; i++ {
			timeBeforeInstall := time.Now()
			logger.Info(fmt.Sprintf("============= Testing (Installing BookInfo %d)%s =============\n", i, name))
			timeoutCtx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
			if err = testcase.InstallBookInfo(timeoutCtx, clientForRunner); err != nil {
				logger.Infof(
					"============= Testing (Install BookInfo Failed)%s =============, Err: \n", name, err.Error(),
				)
				_ = testcase.UninstallBookInfo(clientForRunner)
				continue
			}
			timeAfterInstall := time.Now()
			logger.Infof(
				"============= Testing (BookInfo Installed, Cost(%fs) %s =============\n",
				timeAfterInstall.Sub(timeBeforeInstall).Seconds(), name,
			)
			break
		}

		if err != nil {
			panic(errors.Wrap(err, "test suite failed, install bookinfo error"))
		}

		for i := 0; i < retryTimes; i++ {

			logger.Infof("============= Testing (Wait BookInfo %d)%s =============\n", i, name)

			err = k8sutils.WaitPod(
				clientForRunner.GetClientset(),
				clientForRunner.GetNhctl().Namespace,
				metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "reviews").String()},
				func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
				time.Hour*1,
			)

			err = k8sutils.WaitPod(
				clientForRunner.GetClientset(),
				clientForRunner.GetNhctl().Namespace,
				metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "ratings").String()},
				func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
				time.Hour*1,
			)

			err = k8sutils.WaitPod(
				clientForRunner.GetClientset(),
				clientForRunner.GetNhctl().Namespace,
				metav1.ListOptions{LabelSelector: fields.OneTermEqualSelector("app", "productpage").String()},
				func(i *v1.Pod) bool { return i.Status.Phase == v1.PodRunning },
				time.Hour*1,
			)

			if err == nil {
				break
			}
		}

		if err != nil {
			panic(errors.Wrap(err, "test suite failed, install bookinfo error while wait for pod ready"))
		}
	}

	logger.Infof("============= Testing (Test)%s =============\n", name)

	doneChan := make(chan struct{}, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				t.Clean()
				panic(err)
			}
		}()
		fn(clientForRunner)
		doneChan <- struct{}{}
	}()
	select {
	case <-doneChan:
	case <-time.After(30 * time.Minute):
		timeAfter := time.Now()
		logger.Infof(
			"============= Testing timeout, Cost(%fs) %s =============\n", timeAfter.Sub(timeBefore).Seconds(), name,
		)
		panic(errors.New(fmt.Sprintf("Test %s timeout", name)))
	}

	timeAfter := time.Now()
	logger.Infof(
		"============= Testing done, Cost(%fs) %s =============\n", timeAfter.Sub(timeBefore).Seconds(), name,
	)
}

func (t *T) Clean() {
	if t.CleanFunc != nil {
		t.CleanFunc()
	}
}

func (t *T) Alert() {
	if lastVersion, currentVersion := testcase.GetVersion(); lastVersion != "" && currentVersion != "" {
		if webhook := os.Getenv(util.TestcaseWebhook); webhook != "" {
			s := `{"msgtype":"text","text":{"content":"兼容性测试(%s --> %s)没通过，请相关同学注意啦!",
"mentioned_mobile_list":["18511859195"]}}`
			var req *http.Request
			var err error
			data := strings.NewReader(fmt.Sprintf(s, lastVersion, currentVersion))
			if req, err = http.NewRequest("POST", webhook, data); err != nil {
				log.Info(err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if _, err = http.DefaultClient.Do(req); err != nil {
				log.Info(err)
			}
		}
	}
}

// cli must be kubectl
func (t *T) AlertForImagePull() {
	if webhook := os.Getenv(util.TimeoutWebhook); webhook != "" {
		// some event may not timely
		time.Sleep(time.Minute)

		s1, s2, _ := t.Cli.GetKubectl().RunClusterScope(
			context.TODO(), "get", "events", "-A", "--field-selector", "type!=Normal",
		)

		log.Info("")
		log.Info("")
		log.Info("<< == K8s Events == >>")
		log.Infof("Events show: \n %s%s", s1, s2)
	}
}

func LogsForArchive() {

	for _, l := range log.AllTestLogsLocations() {
		log.Info("")
		log.Info("")
		log.Infof("<< == Final Archive %s == >>", l)
		log.Info(fp.NewFilePath(l).ReadFile())
	}

	log.TestLogger("FINAL").Info("Fin!!!")
}
