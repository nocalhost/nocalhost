/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package kube

import (
	"nocalhost/cmd/nhctl/cmds/common"
	"testing"
)

func TestRunLogs(t *testing.T) {
	common.KubeConfig = "/Users/xinxinhuang/.nh/vscode-plugin/kubeConfigs/ef14063c1455c88e5b39387041bb392d50fd5413"
	common.NameSpace = "nocalhost-aaaa"
	LogOptions.Container = "reviews"
	RunLogs(CmdLogs, []string{"reviews-6446f84d54-xmwbw"})
}
