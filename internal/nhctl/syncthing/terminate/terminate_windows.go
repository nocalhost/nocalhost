// +build windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package terminate

import (
	"github.com/mattn/psutil"
)

func Terminate(pid int, wait bool) error {
	return psutil.TerminateTree(pid, 0)
}
