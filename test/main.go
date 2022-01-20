/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package main

import (
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"nocalhost/test/suite"
	"nocalhost/test/testcase"
	"nocalhost/test/util"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	_ = os.Setenv(_const.EnableFullLogEnvKey, "true")
	start := time.Now()
	var t *suite.T

	if _, ok := os.LookupEnv("LocalTest"); ok {
		// For local test
		_ = os.Setenv(util.CommitId, "test")
		t = suite.NewT("nocalhost", filepath.Join(utils.GetHomePath(), ".kube", "config"), nil)
	} else {
		//_ = os.Setenv(util.CommitId, "test")
		cancelFunc, ns, kubeconfig := suite.Prepare()
		t = suite.NewT(ns, kubeconfig, cancelFunc)
	}

	log.Infof("Init Success, cost: %v", time.Now().Sub(start).Seconds())

	// try to prepare bookinfo image, in case of pull image parallel
	t.RunWithBookInfo(true, "PrepareImage", func(cli runner.Client) {})

	compatibleChan := make(chan interface{}, 1)
	wg := sync.WaitGroup{}

	//DoRun(false, &wg, func() {
	//	t.RunWithBookInfo(false, "TestHook", suite.Hook)
	//})

	DoRun(false, &wg, func() {
		t.RunWithBookInfo(false, "HelmAdaption", suite.HelmAdaption)
	})

	DoRun(false, &wg, func() {
		t.Run("Install", suite.Install)
	})

	DoRun(false, &wg, func() {
		t.Run("Deployment", suite.Deployment)
	})

	DoRun(false, &wg, func() {
		t.Run("Deployment Duplicate", suite.DeploymentDuplicate)
	})

	//DoRun(false, &wg, func() {
	//	t.Run("Deployment Duplicate and Duplicate", testcase.DeploymentDuplicateAndDuplicate)
	//})
	//
	//DoRun(false, &wg, func() {
	//	t.Run("Deployment Replace and Duplicate", testcase.DeploymentReplaceAndDuplicate)
	//})

	DoRun(false, &wg, func() {
		t.Run("Application", suite.Upgrade)
	})

	DoRun(false, &wg, func() {
		t.Run("ProfileAndAssociate", suite.ProfileAndAssociate)
	})

	DoRun(false, &wg, func() {
		t.Run("StatefulSet", suite.StatefulSet)
	})

	//DoRun(false, &wg, func() {
	//	t.Run("StatefulSet Duplicate and Duplicate", testcase.StatefulsetDuplicateAndDuplicate)
	//})
	//
	//DoRun(false, &wg, func() {
	//	t.Run("StatefulSet Replicate and Duplicate", testcase.StatefulsetReplaceAndDuplicate)
	//})

	DoRun(false, &wg, func() {
		t.Run("StatefulSet Duplicate", suite.StatefulSetDuplicate)
	})

	DoRun(false, &wg, func() {
		t.Run("KillSyncthingProcess", suite.KillSyncthingProcess)
	})

	DoRun(false, &wg, func() {
		t.Run("Get", suite.Get)
	})

	DoRun(true, &wg, func() {
		t.Run("Log", suite.TestLog)
	})

	lastVersion, _ := testcase.GetVersion()
	DoRun(lastVersion != "", &wg, func() {
		t.Run("Compatible", suite.Compatible)
		compatibleChan <- "Done"
	})

	wg.Wait()
	log.Infof("All Async Test Done")
	<-compatibleChan

	suite.LogsForArchive()
	log.Infof("Total time: %v", time.Now().Sub(start).Seconds())
	t.Clean()
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
