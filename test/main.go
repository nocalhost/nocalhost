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
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	//_ = os.Setenv("LocalTest", "true")
	_ = os.Setenv(_const.EnableFullLogEnvKey, "true")

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

	log.Infof("Init Success, cost: %v", time.Now().Sub(start).Seconds())

	// try to prepare bookinfo image, in case of pull image parallel
	err := t.RunWithBookInfo(true, "PrepareImage", func(cli runner.Client) error { return nil })
	if err != nil {
		t.AlertForImagePull()
		suite.LogsForArchive()
		//logger.Infof(">>> Panic on suit %s <<<", name)
		//logger.Infof("Error before recover %v", err)
		t.Clean()
		t.Alert()
		panic(err)
	}

	compatibleChan := make(chan interface{}, 1)
	wg := sync.WaitGroup{}

	//DoRun(false, &wg, func() {
	//	t.RunWithBookInfo(false, "TestHook", suite.Hook)
	//})

	DoRun(false, &wg, func() error {
		return t.RunWithBookInfo(false, "HelmAdaption", suite.HelmAdaption)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Install", suite.Install)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Deployment", suite.Deployment)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Deployment Duplicate", suite.DeploymentDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Deployment Duplicate and Duplicate", testcase.DeploymentDuplicateAndDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Deployment Replace and Duplicate", testcase.DeploymentReplaceAndDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Application", suite.Upgrade)
	})

	DoRun(false, &wg, func() error {
		return t.Run("ProfileAndAssociate", suite.ProfileAndAssociate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("StatefulSet", suite.StatefulSet)
	})

	DoRun(false, &wg, func() error {
		return t.Run("StatefulSet Duplicate and Duplicate", testcase.StatefulsetDuplicateAndDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("StatefulSet Replicate and Duplicate", testcase.StatefulsetReplaceAndDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("StatefulSet Duplicate", suite.StatefulSetDuplicate)
	})

	DoRun(false, &wg, func() error {
		return t.Run("KillSyncthingProcess", suite.KillSyncthingProcess)
	})

	DoRun(false, &wg, func() error {
		return t.Run("Get", suite.Get)
	})

	DoRun(true, &wg, func() error {
		return t.Run("Log", suite.TestLog)
	})

	if _, ok := os.LookupEnv("LocalTest"); !ok {
		lastVersion, _ := testcase.GetVersion()
		DoRun(lastVersion != "", &wg, func() error {
			err := t.Run("Compatible", suite.Compatible)
			if err != nil {
				return err
			}
			compatibleChan <- "Done"
			return nil
		})
	}

	wg.Wait()
	log.Infof("All Async Test Done")
	<-compatibleChan

	suite.LogsForArchive()
	log.Infof("Total time: %v", time.Now().Sub(start).Seconds())
	t.Clean()
}

func DoRun(doAfterWgDone bool, wg *sync.WaitGroup, do func() error) {
	if !doAfterWgDone {
		wg.Add(1)
		go func() {
			err := do()
			if err != nil {
				panic(err)
			}
			wg.Done()
		}()
	} else {
		go func() {
			wg.Wait()
			err := do()
			if err != nil {
				panic(err)
			}
		}()
	}
}
