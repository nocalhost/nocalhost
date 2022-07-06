/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

import (
	"fmt"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"testing"
)

func TestHandleGetResourceInfoRequest(t *testing.T) {
	i, err := HandleGetResourceInfoRequest(&command.GetResourceInfoCommand{
		CommandType:  "",
		ClientStack:  "",
		KubeConfig:   filepath.Join(utils.GetHomePath(), ".kube", "config"),
		Namespace:    "nocalhost-test",
		AppName:      "bookinfo",
		Resource:     "deployment",
		ResourceName: "",
		Label:        nil,
		ShowHidden:   false,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(i)
}
