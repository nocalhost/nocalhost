// +build !windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package terminate

import (
	"os"
	"strings"
)

const (
	syncthing = "syncthing"
)

func Terminate(pid int, wait bool) error {
	// if typeName=syncthing, it should use proc.Signal(os.Inte)
	proc := os.Process{Pid: pid}
	if err := proc.Kill(); err != nil {
		if strings.Contains(err.Error(), "process already finished") {
			return nil
		}
		return err
	}
	if wait {
		_, _ = proc.Wait()
	}
	return nil

	// dev port-forward and sync port-forward can only use proc.Kill()
	//if err := proc.Kill(); err != nil {
	//	if strings.Contains(err.Error(), "process already finished") {
	//		return nil
	//	}
	//	return err
	//}

	//if wait {
	//	defer proc.Wait() // nolint: errcheck
	//}
	//return nil
}
