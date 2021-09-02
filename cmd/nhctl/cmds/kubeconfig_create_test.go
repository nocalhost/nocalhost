/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"testing"
)

func TestGenKubeconfig(t *testing.T) {
	GenKubeconfig(filepath.Join(utils.GetHomePath(), ".kube", "devcloud-config"), "bbba")
}
