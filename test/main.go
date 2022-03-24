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
	t.RunWithBookInfo(false, "HelmAdaption", suite.HelmAdaption)
	t.Run("Install", suite.Install)
	t.Run("Deployment", suite.Deployment)
	t.Run("Deployment Duplicate", suite.DeploymentDuplicate)
	t.Run("Application", suite.Upgrade)
	t.Run("ProfileAndAssociate", suite.ProfileAndAssociate)
	t.Run("StatefulSet", suite.StatefulSet)
	t.Run("StatefulSet Duplicate", suite.StatefulSetDuplicate)
	t.Run("KillSyncthingProcess", suite.KillSyncthingProcess)
	t.Run("Get", suite.Get)
	t.Run("Log", suite.TestLog)
	if lastVersion, _ := testcase.GetVersion(); lastVersion != "" {
		t.Run("Compatible", suite.Compatible)
	}
	log.Infof("All Test Done")
	log.Infof("Total time: %v", time.Now().Sub(start).Seconds())
	t.Clean()
}
