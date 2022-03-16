/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/profile"
)

// EnterPodTerminal Try to use shell defined in devContainerShell to enter pod's terminal
// If devContainerShell is not defined or shell defined in devContainerShell failed to enter terminal, use /bin/sh
// If container not specified, the first container will be used
func (c *Controller) EnterPodTerminal(podName, container, shell, banner string) error {
	pod := podName

	var devContainerName = container
	var cfg *profile.ContainerDevConfig

	pf := c.Config()
	if pf != nil {
		devConfig := pf.GetContainerDevConfigOrDefault(container)
		cfg = devConfig
	}

	if cfg != nil {
		if shell == "" {
			shell = cfg.Shell
		}

		if c.IsInDevMode() {
			if cfg.DevContainerName != "" {
				devContainerName = cfg.DevContainerName
			} else {
				devContainerName = _const.NocalhostDefaultDevContainerName
			}
		}
	}

	cmd := "(zsh || bash || sh)"
	if shell != "" {
		cmd = fmt.Sprintf("(%s || zsh || bash || sh)", shell)
	}
	return c.Client.ExecShell(pod, devContainerName, cmd, banner)
}
