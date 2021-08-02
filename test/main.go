/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package main

import (
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/suite"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	//_ = os.Setenv("LocalTest", "true")

	start := time.Now()

	var t *suite.T

	if _, ok := os.LookupEnv("LocalTest"); ok {
		// For local test
		t = suite.NewT("nocalhost", filepath.Join(utils.GetHomePath(), ".kube", "config"), nil)
		_ = os.Setenv("commit_id", "test")
	} else {
		cancelFunc, ns, kubeconfig := suite.Prepare()
		t = suite.NewT(ns, kubeconfig, cancelFunc)
	}
	// try to prepare bookinfo image, in case of pull image parallel
	t.RunWithBookInfo(true, "prepare image", func(cli runner.Client) {})

	compatibleChan := make(chan interface{}, 1)
	wg := sync.WaitGroup{}

	DoRun(false, &wg, func() {
		t.RunWithBookInfo(false, "helm-adaption", suite.HelmAdaption)
	})

	DoRun(false, &wg, func() {
		t.Run("install", suite.Install)
	})

	DoRun(false, &wg, func() {
		t.Run("deployment", suite.Deployment)
	})

	DoRun(false, &wg, func() {
		t.Run("application", suite.Upgrade)
	})

	DoRun(false, &wg, func() {
		t.Run("statefulSet", suite.StatefulSet)
	})

	DoRun(false, &wg, func() {
		t.Run("remove syncthing pid file manually", suite.KillSyncthingProcess)
	})

	DoRun(false, &wg, func() {
		t.Run("Get", suite.Get)
	})

	DoRun(true, &wg, func() {
		t.Run("compatible", suite.Compatible)
		compatibleChan <- "Done"
	})

	wg.Wait()
	log.Infof("All Async Test Done")
	<-compatibleChan

	t.Clean(false)

	log.Infof("Total time: %v", time.Now().Sub(start).Seconds())
}

func DoRun(doAfterWgDone bool, wg *sync.WaitGroup, do func()) {
	if !doAfterWgDone {
		wg.Add(1)
		go func() {
			do()
			wg.Done()
		}()
	} else {
		go func() {
			wg.Wait()
			do()
		}()
	}
}
