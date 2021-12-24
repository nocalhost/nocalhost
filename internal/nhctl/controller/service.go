/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

// Controller presents a k8s controller
// https://kubernetes.io/docs/concepts/architecture/controller
type Controller struct {
	NameSpace   string
	AppName     string
	Name        string
	Identifier  string
	DevModeType profile.DevModeType
	Type        base.SvcType
	Client      *clientgoutils.ClientGoUtils
	AppMeta     *appmeta.ApplicationMeta
	config      *profile.ServiceConfigV2
}

func NewController(ns, name, appName, identifier string, svcType base.SvcType,
	client *clientgoutils.ClientGoUtils, appMeta *appmeta.ApplicationMeta) (*Controller, error) {
	c := &Controller{
		NameSpace:  ns,
		AppName:    appName,
		Name:       name,
		Type:       svcType,
		Client:     client,
		AppMeta:    appMeta,
		Identifier: identifier,
	}
	c.DevModeType = c.GetCurrentDevModeType()

	a := c.GetAppConfig().GetSvcConfigS(c.Name, c.Type)
	c.config = &a

	return c, nil
}

// IsInReplaceDevMode return true if under dev starting or start complete
func (c *Controller) IsInReplaceDevMode() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Identifier, c.Type, profile.ReplaceDevMode) != appmeta.NONE
}

func (c *Controller) IsInReplaceDevModeStarting() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Identifier, c.Type, profile.ReplaceDevMode) == appmeta.STARTING
}

func (c *Controller) IsInDuplicateDevMode() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Identifier, c.Type, profile.DuplicateDevMode) != appmeta.NONE
}

func (c *Controller) IsInDuplicateDevModeStarting() bool {
	return c.AppMeta.CheckIfSvcDeveloping(c.Name, c.Identifier, c.Type, profile.DuplicateDevMode) == appmeta.STARTING
}

func (c *Controller) IsInDevMode() bool {
	return c.IsInDuplicateDevMode() || c.IsInReplaceDevMode()
}

func (c *Controller) IsInDevModeStarting() bool {
	return c.IsInDuplicateDevModeStarting() || c.IsInReplaceDevModeStarting()
}

// IsProcessor Check if service is developing in this device
func (c *Controller) IsProcessor() bool {
	return c.AppMeta.SvcDevModePossessor(c.Name, c.Type, c.Identifier, profile.DuplicateDevMode) || c.AppMeta.SvcDevModePossessor(c.Name, c.Type, c.Identifier, profile.ReplaceDevMode)
}

func (c *Controller) GetCurrentDevModeType() profile.DevModeType {
	return c.AppMeta.GetCurrentDevModeTypeOfWorkload(c.Name, c.Type, c.Identifier)
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

func (c *Controller) GetOriginalContainers() ([]v1.Container, error) {
	return GetOriginalContainers(c.Client, c.Type, c.Name)
}

func GetOriginalContainers(client *clientgoutils.ClientGoUtils, workloadType base.SvcType, workloadName string) ([]v1.Container, error) {
	var podSpec v1.PodSpec
	switch workloadType {
	case base.Deployment:
		d, err := client.GetDeployment(workloadName)
		if err != nil {
			return nil, err
		}
		if len(d.Annotations) > 0 {
			if osj, ok := d.Annotations[OriginSpecJson]; ok {
				d.Spec = appsv1.DeploymentSpec{}
				if err = json.Unmarshal([]byte(osj), &d.Spec); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = d.Spec.Template.Spec
	case base.StatefulSet:
		s, err := client.GetStatefulSet(workloadName)
		if err != nil {
			return nil, err
		}
		if len(s.Annotations) > 0 {
			if osj, ok := s.Annotations[OriginSpecJson]; ok {
				s.Spec = appsv1.StatefulSetSpec{}
				if err = json.Unmarshal([]byte(osj), &s.Spec); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = s.Spec.Template.Spec
	case base.DaemonSet:
		d, err := client.GetDaemonSet(workloadName)
		if err != nil {
			return nil, err
		}
		if len(d.Annotations) > 0 {
			if osj, ok := d.Annotations[OriginSpecJson]; ok {
				d.Spec = appsv1.DaemonSetSpec{}
				if err = json.Unmarshal([]byte(osj), &d.Spec); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = d.Spec.Template.Spec
	case base.Job:
		j, err := client.GetJobs(workloadName)
		if err != nil {
			return nil, err
		}
		if len(j.Annotations) > 0 {
			if osj, ok := j.Annotations[OriginSpecJson]; ok {
				j.Spec = batchv1.JobSpec{}
				if err = json.Unmarshal([]byte(osj), &j.Spec); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = j.Spec.Template.Spec
	case base.CronJob:
		j, err := client.GetCronJobs(workloadName)
		if err != nil {
			return nil, err
		}
		if len(j.Annotations) > 0 {
			if osj, ok := j.Annotations[OriginSpecJson]; ok {
				j.Spec = batchv1beta1.CronJobSpec{}
				if err = json.Unmarshal([]byte(osj), &j.Spec); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = j.Spec.JobTemplate.Spec.Template.Spec
	case base.Pod:
		p, err := client.GetPod(workloadName)
		if err != nil {
			return nil, err
		}
		if len(p.Annotations) > 0 {
			if osj, ok := p.Annotations[originalPodDefine]; ok {
				p.Spec = v1.PodSpec{}
				if err = json.Unmarshal([]byte(osj), p); err != nil {
					return nil, errors.Wrap(err, "")
				}
			}
		}
		podSpec = p.Spec
	}

	return podSpec.Containers, nil
}

func (c *Controller) GetTypeMeta() (metav1.TypeMeta, error) {
	switch c.Type {
	case base.Deployment:
		return metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		}, nil
	case base.StatefulSet:
		return metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		}, nil
		//return appsv1.StatefulSet{}.TypeMeta, nil
	case base.DaemonSet:
		return metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		}, nil
		//return appsv1.DaemonSet{}.TypeMeta, nil
	case base.Job:
		return metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		}, nil
		//return batchv1.Job{}.TypeMeta, nil
	case base.CronJob:
		return metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1beta1",
		}, nil
		//return batchv1beta1.CronJob{}.TypeMeta, nil
	case base.Pod:
		return metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		}, nil
	default:
		return metav1.TypeMeta{}, errors.New("unsupported controller type")
	}
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
	profileV2, err := profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName, c.AppMeta.NamespaceId)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	if err := modify(profileV2.SvcProfileV2(c.Name, c.Type.String())); err != nil {
		return err
	}
	profileV2.GenerateIdentifierIfNeeded()
	return profileV2.Save()
}

func (c *Controller) GetName() string {
	return c.Name
}

func (c *Controller) getDuplicateLabelsMap() (map[string]string, error) {

	labelsMap := map[string]string{
		IdentifierKey:             c.Identifier,
		OriginWorkloadNameKey:     c.Name,
		OriginWorkloadTypeKey:     string(c.Type),
		_const.DevWorkloadIgnored: "true",
	}
	return labelsMap, nil
}

func (c *Controller) getDuplicateResourceName() string {
	return strings.Join([]string{c.Name, string(c.Type), c.Identifier[0:5], strconv.Itoa(int(time.Now().Unix()))}, "-")
}

func (c *Controller) patchAfterDevContainerReplaced(containerName, resourceType, resourceName string) {
	for _, patch := range c.config.GetContainerDevConfigOrDefault(containerName).Patches {
		log.Infof("Patching %s", patch.Patch)
		if err := c.Client.Patch(resourceType, resourceName, patch.Patch, patch.Type); err != nil {
			log.WarnE(err, "")
		}
	}
	<-time.Tick(time.Second)
}
