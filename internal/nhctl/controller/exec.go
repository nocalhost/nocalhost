/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package controller

import (
	"fmt"
)

// EnterPodTerminal Try to use shell defined in devContainerShell to enter pod's terminal
// If devContainerShell is not defined or shell defined in devContainerShell failed to enter terminal, use /bin/sh
// If container not specified, the first container will be used
func (c *Controller) EnterPodTerminal(podName, container string, shell string) error {
	pod := podName

	if shell == "" {
		profile, _ := c.GetProfile()
		if profile != nil {
			devConfig := profile.GetContainerDevConfigOrDefault(container)
			if devConfig != nil {
				shell = devConfig.Shell
			}
		}
	}

	cmd := "(zsh || bash || sh)"
	if shell != "" {
		cmd = fmt.Sprintf("(%s || zsh || bash || sh)", shell)
	}
	return c.Client.ExecShell(pod, container, cmd)
}
