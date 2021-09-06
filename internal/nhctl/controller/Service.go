/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
)

// Controller presents a k8s controller
// https://kubernetes.io/docs/concepts/architecture/controller
type Controller struct {
	NameSpace string
	AppName   string
	Name      string
	Type      base.SvcType
	Client    *clientgoutils.ClientGoUtils
	AppMeta   *appmeta.ApplicationMeta
}

// IsInDevMode return true if under dev starting or start complete
func (c *Controller) IsInDevMode() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Type) != appmeta.NONE
}

func (c *Controller) IsProcessor() bool {
	appProfile, err := c.GetAppProfile() // todo: move Identifier to directory
	if err != nil {
		return false
	}
	return c.AppMeta.SvcDevModePossessor(c.Name, c.Type, appProfile.Identifier)
}

func CheckIfControllerTypeSupport(t string) bool {
	tt := base.SvcType(t)
	if tt == base.Deployment || tt == base.StatefulSet || tt == base.DaemonSet || tt == base.Job ||
		tt == base.CronJob || tt == base.Pod {
		return true
	}
	return false
}

func (c *Controller) CheckIfExist() (bool, error) {
	var err error
	switch c.Type {
	case base.Deployment:
		_, err = c.Client.GetDeployment(c.Name)
	case base.StatefulSet:
		_, err = c.Client.GetStatefulSet(c.Name)
	case base.DaemonSet:
		_, err = c.Client.GetDaemonSet(c.Name)
	case base.Job:
		_, err = c.Client.GetJobs(c.Name)
	case base.CronJob:
		_, err = c.Client.GetCronJobs(c.Name)
	case base.Pod:
		_, err = c.Client.GetPod(c.Name)
	default:
		return false, errors.New("unsupported controller type")
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Controller) GetContainerImage(container string) (string, error) {
	var podSpec v1.PodSpec
	switch c.Type {
	case base.Deployment:
		d, err := c.Client.GetDeployment(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = d.Spec.Template.Spec
	case base.StatefulSet:
		s, err := c.Client.GetStatefulSet(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = s.Spec.Template.Spec
	case base.DaemonSet:
		d, err := c.Client.GetDaemonSet(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = d.Spec.Template.Spec
	case base.Job:
		j, err := c.Client.GetJobs(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = j.Spec.Template.Spec
	case base.CronJob:
		j, err := c.Client.GetCronJobs(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = j.Spec.JobTemplate.Spec.Template.Spec
	case base.Pod:
		p, err := c.Client.GetPod(c.Name)
		if err != nil {
			return "", err
		}
		podSpec = p.Spec
	}

	for _, c := range podSpec.Containers {
		if c.Name == container {
			return c.Image, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Container %s not found", container))
}

func (c *Controller) GetContainers() ([]v1.Container, error) {
	var podSpec v1.PodSpec
	switch c.Type {
	case base.Deployment:
		d, err := c.Client.GetDeployment(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = d.Spec.Template.Spec
	case base.StatefulSet:
		s, err := c.Client.GetStatefulSet(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = s.Spec.Template.Spec
	case base.DaemonSet:
		d, err := c.Client.GetDaemonSet(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = d.Spec.Template.Spec
	case base.Job:
		j, err := c.Client.GetJobs(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = j.Spec.Template.Spec
	case base.CronJob:
		j, err := c.Client.GetCronJobs(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = j.Spec.JobTemplate.Spec.Template.Spec
	case base.Pod:
		p, err := c.Client.GetPod(c.Name)
		if err != nil {
			return nil, err
		}
		podSpec = p.Spec
	}

	return podSpec.Containers, nil
}

func (c *Controller) GetDescription() *profile.SvcProfileV2 {
	appProfile, err := c.GetAppProfile()
	if err != nil {
		return nil
	}
	svcProfile := appProfile.SvcProfileV2(c.Name, string(c.Type))
	if svcProfile != nil {
		appmeta.FillingExtField(svcProfile, c.AppMeta, c.AppName, c.NameSpace, appProfile.Identifier)
		return svcProfile
	}
	return nil
}

func (c *Controller) UpdateSvcProfile(modify func(*profile.SvcProfileV2) error) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	if err := modify(profileV2.SvcProfileV2(c.Name, c.Type.String())); err != nil {
		return err
	}
	return profileV2.Save()
}

func (c *Controller) UpdateProfile(modify func(*profile.AppProfileV2, *profile.SvcProfileV2) error) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	if err := modify(profileV2, profileV2.SvcProfileV2(c.Name, c.Type.String())); err != nil {
		return err
	}
	return profileV2.Save()
}
