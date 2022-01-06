/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"testing"
)

func TestRunTviewApplication(t *testing.T) {
	RunTviewApplication()
}

func TestSyncBuilder_Sync(t *testing.T) {
	log.Info("init log")
	sbd := SyncBuilder{&strings.Builder{}, func(p []byte) (int, error) {
		fmt.Println("tttt:" + string(p))
		return 0, nil
	}}

	log.RedirectionDefaultLogger(&sbd)
	log.Info("aaaa")
	//log.Info("syncing")
}
