/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package log

import "testing"

func TestPWarn(t *testing.T) {
	PWarn("This is a warning")
	PWarnf("This is a warning in %d/%d/%d", 2021, 07, 25)
}
