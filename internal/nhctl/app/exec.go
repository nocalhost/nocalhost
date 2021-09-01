/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	v1 "k8s.io/api/core/v1"
)

func (a *Application) Exec(pod v1.Pod, container string, commands []string) error {
	var name string
	// if container arguments are available, using container arguments
	// else if found nocalhost-dev container, using nocalhost-dev
	// else return error
	for _, c := range pod.Spec.Containers {
		if c.Name == "nocalhost-dev" {
			name = c.Name
		}
		if c.Name == container {
			name = container
			break
		}
	}
	return a.client.Exec(pod.Name, name, commands)
}
