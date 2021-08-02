/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) Exec(svcName string, container string, commands []string) error {
	podList, err := a.client.ListPodsByDeployment(svcName)
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		log.Warnf("the number of pods of %s is not 1 ???", svcName)
		return errors.New(fmt.Sprintf("the number of pods of %s is not 1 ???", svcName))
	}
	pod := podList.Items[0].Name
	var name string
	for _, c := range podList.Items[0].Spec.Containers {
		if c.Name == "nocalhost-dev" {
			name = c.Name
		}
		if c.Name == container {
			name = container
			break
		}
	}
	if len(name) == 0 {
		return errors.New(fmt.Sprintf("container: %s not found", name))
	}
	return a.client.Exec(pod, name, commands)
}
