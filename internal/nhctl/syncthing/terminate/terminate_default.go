// +build !windows

/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package terminate

import (
	"os"
	"strings"
)

const (
	syncthing = "syncthing"
)

func Terminate(pid int, wait bool, typeName string) error {
	// if typeName=syncthing, it should use proc.Signal(os.Inte)
	proc := os.Process{Pid: pid}
	if typeName == syncthing {
		if err := proc.Signal(os.Interrupt); err != nil {
			if strings.Contains(err.Error(), "process already finished") {
				return nil
			}
			return err
		}
		return nil
	}

	// dev port-forward and sync port-forward can only use proc.Kill()
	if err := proc.Kill(); err != nil {
		if strings.Contains(err.Error(), "process already finished") {
			return nil
		}
		return err
	}

	if wait {
		defer proc.Wait() // nolint: errcheck
	}
	return nil
}
