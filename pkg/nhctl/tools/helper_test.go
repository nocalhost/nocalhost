/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package tools

import (
	"context"
	"testing"
)

func TestExecCommand(t *testing.T) {
	ExecCommand(context.TODO(), true, false, "sleep", "3600")
}
