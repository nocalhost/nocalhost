/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Controller presents a k8s controller
// https://kubernetes.io/docs/concepts/architecture/controller
type Controller struct {
	NameSpace        string
	AppName          string
	Name             string
	Identifier       string
	DevModeType      profile.DevModeType
	Type             base.SvcType
	Client           *clientgoutils.ClientGoUtils
	AppMeta          *appmeta.ApplicationMeta
	config           *profile.ServiceConfigV2
	DevModeAction    base.DevModeAction
	devModePodLabels kblabels.Set
}

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
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
	da, err := nocalhost.GetDevModeActionBySvcType(svcType)
	if err == nil {
		c.DevModeAction = *da
	}

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
	return c.AppMeta.SvcDevModePossessor(
		c.Name, c.Type, c.Identifier, profile.DuplicateDevMode,
	) || c.AppMeta.SvcDevModePossessor(c.Name, c.Type, c.Identifier, profile.ReplaceDevMode)
}

func (c *Controller) GetCurrentDevModeType() profile.DevModeType {
	return c.AppMeta.GetCurrentDevModeTypeOfWorkload(c.Name, c.Type, c.Identifier)
}

//func CheckIfControllerTypeSupport(t string) bool {
//	tt := base.SvcType(t)
//	if tt == base.Deployment || tt == base.StatefulSet || tt == base.DaemonSet || tt == base.Job ||
//		tt == base.CronJob || tt == base.Pod || tt == base.CloneSetV1Alpha1 {
//		return true
//	}
//	return false
//}

func (c *Controller) CheckIfExist() (bool, error) {
	_, err := c.GetUnstructured()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Controller) GetOriginalContainers() ([]v1.Container, error) {
	return GetOriginalContainers(c.Client, c.Type, c.Name, c.DevModeAction.PodTemplatePath)
}

func GetOriginalContainers(client *clientgoutils.ClientGoUtils, workloadType base.SvcType, workloadName, path string) ([]v1.Container, error) {
	//var podSpec v1.PodSpec
	um, err := client.GetUnstructured(string(workloadType), workloadName)
	if err != nil {
		return nil, err
	}

	var originalUm *unstructured.Unstructured
	od, err := GetAnnotationFromUnstructured(um, _const.OriginWorkloadDefinition)
	if err == nil {
		originalUm, _ = client.GetUnstructuredFromString(od)
	}
	if originalUm != nil {
		um = originalUm
	}

	pt, err := GetPodTemplateFromSpecPath(path, um.Object)
	if err != nil {
		return nil, err
	}
	return pt.Spec.Containers, nil
}

//func GetOriginalContainers(name string, workloadType base.SvcType, client *clientgoutils.ClientGoUtils) ([]v1.Container, error) {
//
//	//switch workloadType {
//	//case base.Deployment:
//	//	d, err := client.GetDeployment(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(d.Annotations) > 0 {
//	//		if osj, ok := d.Annotations[OriginSpecJson]; ok {
//	//			d.Spec = appsv1.DeploymentSpec{}
//	//			if err = json.Unmarshal([]byte(osj), &d.Spec); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = d.Spec.Template.Spec
//	//case base.StatefulSet:
//	//	s, err := client.GetStatefulSet(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(s.Annotations) > 0 {
//	//		if osj, ok := s.Annotations[OriginSpecJson]; ok {
//	//			s.Spec = appsv1.StatefulSetSpec{}
//	//			if err = json.Unmarshal([]byte(osj), &s.Spec); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = s.Spec.Template.Spec
//	//case base.DaemonSet:
//	//	d, err := client.GetDaemonSet(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(d.Annotations) > 0 {
//	//		if osj, ok := d.Annotations[OriginSpecJson]; ok {
//	//			d.Spec = appsv1.DaemonSetSpec{}
//	//			if err = json.Unmarshal([]byte(osj), &d.Spec); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = d.Spec.Template.Spec
//	//case base.Job:
//	//	j, err := client.GetJobs(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(j.Annotations) > 0 {
//	//		if osj, ok := j.Annotations[OriginSpecJson]; ok {
//	//			j.Spec = batchv1.JobSpec{}
//	//			if err = json.Unmarshal([]byte(osj), &j.Spec); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = j.Spec.Template.Spec
//	//case base.CronJob:
//	//	j, err := client.GetCronJobs(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(j.Annotations) > 0 {
//	//		if osj, ok := j.Annotations[OriginSpecJson]; ok {
//	//			j.Spec = batchv1beta1.CronJobSpec{}
//	//			if err = json.Unmarshal([]byte(osj), &j.Spec); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = j.Spec.JobTemplate.Spec.Template.Spec
//	//case base.Pod:
//	//	p, err := client.GetPod(workloadName)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//	if len(p.Annotations) > 0 {
//	//		if osj, ok := p.Annotations[originalPodDefine]; ok {
//	//			p.Spec = v1.PodSpec{}
//	//			if err = json.Unmarshal([]byte(osj), p); err != nil {
//	//				return nil, errors.Wrap(err, "")
//	//			}
//	//		}
//	//	}
//	//	podSpec = p.Spec
//	//}
//
//	return pt.Spec.Containers, nil
//}}

func (c *Controller) GetTypeMeta() (metav1.TypeMeta, error) {
	um, err := c.GetUnstructured()
	if err != nil {
		return metav1.TypeMeta{}, err
	}

	result := metav1.TypeMeta{}

	k := um.GetKind()
	if k == "" {
		return result, errors.New("Can not find kind")
	}
	result.Kind = k

	a := um.GetAPIVersion()
	if a == "" {
		return result, errors.New("Can not find apiVersion")
	}
	result.APIVersion = a
	return result, nil
}

func (c *Controller) GetContainerImage(container string) (string, error) {
	cs, err := c.GetContainers()
	if err != nil {
		return "", err
	}

	for _, c := range cs {
		if c.Name == container {
			return c.Image, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Container %s not found", container))
}

func (c *Controller) GetContainers() ([]v1.Container, error) {
	um, err := c.GetUnstructured()
	if err != nil {
		return nil, err
	}

	pt, err := GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object)
	if err != nil {
		return nil, err
	}
	return pt.Spec.Containers, nil
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
	return strings.Join([]string{c.Name, c.Identifier[0:5], strconv.Itoa(int(time.Now().Unix()))}, "-")
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

//func genDevContainerPatches(podSpec *v1.PodSpec, path, originalSpecJson string) []profile.PatchItem {
//
//	jsonPatches := make([]jsonPatch, 0)
//	jsonPatches = append(jsonPatches, jsonPatch{
//		Op:    "replace",
//		Path:  path,
//		Value: podSpec,
//	})
//
//	//jsonPatches = append(jsonPatches, jsonPatch{
//	//	Op:    "add",
//	//	Path:  fmt.Sprintf("/metadata/annotations/%s", _const.OriginWorkloadDefinition),
//	//	Value: originalSpecJson,
//	//})
//	m := map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{_const.OriginWorkloadDefinition: originalSpecJson}}}
//
//	mBytes, _ := json.Marshal(m)
//	bys, _ := json.Marshal(jsonPatches)
//	result := make([]profile.PatchItem, 0)
//	result = append(result, profile.PatchItem{Patch: string(mBytes), Type: "strategic"})
//	result = append(result, profile.PatchItem{Patch: string(bys), Type: "json"})
//
//	return result
//}

func (c *Controller) getGeneratedDeploymentName() string {
	t := strings.ToLower(strings.Split(string(c.Type), ".")[0])
	id, _ := utils.GetShortUuid()
	return fmt.Sprintf("%s-%s-devmode-%s", c.GetName(), t, id)
}

func (c *Controller) getGeneratedDeploymentLabels() map[string]string {
	return map[string]string{
		_const.DevWorkloadIgnored:     "true",
		"dev.nocalhost/workload-type": string(c.Type),
		"dev.nocalhost/workload-name": c.Name,
	}
}

func (c *Controller) PatchDevModeManifest(ctx context.Context, ops *model.DevStartOptions) error {
	c.Client.Context(ctx)

	unstructuredObj, err := c.GetUnstructured()
	if err != nil {
		return err
	}

	RemoveUselessInfo(unstructuredObj)

	var originalSpecJson []byte
	if originalSpecJson, err = json.Marshal(unstructuredObj); err != nil {
		return errors.WithStack(err)
	}

	log.Infof("Scale %s(%s) to 1", c.Name, c.Type.String())
	for _, item := range c.DevModeAction.ScalePatches {
		log.Infof("Patching %s", item.Patch)
		if err := c.Client.Patch(c.Type.String(), c.Name, item.Patch, item.Type); err != nil {
			return err
		}
	}
	log.Info("Scale success")

	podTemplate, err := GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, unstructuredObj.Object)
	if err != nil {
		return err
	}

	podSpec := &podTemplate.Spec

	devContainer, sideCarContainer, devModeVolumes, err :=
		c.genContainersAndVolumes(podSpec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(podSpec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	m := map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{_const.OriginWorkloadDefinition: string(originalSpecJson)}}}
	mBytes, _ := json.Marshal(m)
	if err = c.Client.Patch(c.Type.String(), c.Name, string(mBytes), "merge"); err != nil {
		return err
	}
	log.Info("Original manifest recorded")

	if !c.DevModeAction.Create {
		log.Info("Patching development container...")

		specPath := c.DevModeAction.PodTemplatePath + "/spec"
		jsonPatches := make([]jsonPatch, 0)
		jsonPatches = append(
			jsonPatches, jsonPatch{
				Op:    "replace",
				Path:  specPath,
				Value: podSpec,
			},
		)
		bys, _ := json.Marshal(jsonPatches)

		if err = c.Client.Patch(c.Type.String(), c.Name, string(bys), "json"); err != nil {
			return err
		}
		c.patchAfterDevContainerReplaced(ops.Container, c.Type.String(), c.Name)
	} else {
		// Some workload's pod may not have labels, such as cronjob, we need to give it one
		if len(podTemplate.Labels) == 0 {
			podTemplate.Labels = c.getGeneratedDeploymentLabels()
		}
		generatedDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   c.getGeneratedDeploymentName(),
				Labels: c.getGeneratedDeploymentLabels(),
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: podTemplate.Labels,
				},
				Template: *podTemplate,
			},
		}
		generatedDeployment.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyAlways
		if _, err = c.Client.CreateDeployment(generatedDeployment); err != nil {
			return err
		}
		c.patchAfterDevContainerReplaced(ops.Container, "deployment", generatedDeployment.Name)
	}

	delete(podTemplate.Labels, "pod-template-hash")
	c.devModePodLabels = podTemplate.Labels

	gvr := c.Client.ResourceFor("pod", false)
	gvk, gvkErr := c.Client.KindFor(gvr)
	if gvkErr != nil {
		log.Infof(
			"(Can Be Ignore, does not affect the actual function) "+
				"Error to find workload's GVK "+
				"(Event Watcher when dev start will be invalid), Error:%s", gvk,
		)
	}

	log.Infof("\nNow waiting dev mode to start...\n")

	// start a watcher for dev pod until it running
	//
	// print it's status & container status
	// the printer used to help filter the same content
	//
	// if a pod is being running, close the quitChan to stop block and informer
	//
	printer := utils.NewPrinter(
		func(s string) {
			log.Infof(s)
		},
	)

	var currentPod atomic.Value

	quitChan := watcher.NewSimpleWatcher(
		c.Client,
		"pod",
		c.devModePodLabels.AsSelector().String(),
		func(key string, object interface{}, quitChan chan struct{}) {
			if us, ok := object.(*unstructured.Unstructured); ok {

				var pod v1.Pod
				if err := runtime.DefaultUnstructuredConverter.
					FromUnstructured(us.UnstructuredContent(), &pod); err == nil {

					if containerStatusForDevPod(
						&pod, func(status string, err error) {
							printer.ChangeContent(status)
						},
					) {
						currentPod.Store(pod.Name)
					}

					if _, err := findDevPodName(pod); err == nil {
						close(quitChan)
					}
					return
				}
			}
		},
		nil,
	)

	if gvkErr == nil && !gvk.Empty() {
		apiVersion, kind := gvk.ToAPIVersionAndKind()

		c := watcher.NewSimpleWatcher(
			c.Client,
			"event",
			"",
			func(key string, object interface{}, quitChan chan struct{}) {
				if us, ok := object.(*unstructured.Unstructured); ok {

					var event v1.Event
					if err := runtime.DefaultUnstructuredConverter.
						FromUnstructured(us.UnstructuredContent(), &event); err == nil {

						podName := currentPod.Load()

						if podName == nil ||
							event.Type == "Normal" ||
							event.InvolvedObject.Kind != kind ||
							event.InvolvedObject.APIVersion != apiVersion ||
							podName.(string) != event.InvolvedObject.Name {
							return
						}

						printer.ChangeContent(fmt.Sprintf("### Notable events: %s: %s ###", event.Reason, event.Message))
						return
					}
				}
			},
			nil,
		)
		defer close(c)
	}

	<-quitChan
	return nil
}

func (c *Controller) CheckDevModePodIsRunning() (string, error) {
	pods, err := c.Client.Labels(c.devModePodLabels).ListPods()
	if err != nil {
		return "", err
	}
	return findDevPodName(pods...)
}

// ReplaceDuplicateModeImage Create a duplicate deployment instead of replacing image
func (c *Controller) ReplaceDuplicateModeImage(ctx context.Context, ops *model.DevStartOptions) error {
	c.Client.Context(ctx)

	um, err := c.GetUnstructured()
	if err != nil {
		return err
	}

	RemoveUselessInfo(um)

	if c.IsInReplaceDevMode() {
		od, err := GetAnnotationFromUnstructured(um, _const.OriginWorkloadDefinition)
		if err != nil {
			return err
		}

		if um, err = c.Client.GetUnstructuredFromString(od); err != nil {
			return err
		}
	}

	var ps *v1.PodTemplateSpec
	if !c.DevModeAction.Create {
		if err = c.PatchDuplicateInfo(um.Object); err != nil {
			return err
		}
		if ps, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(&ps.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(&ps.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

		// Set podTemplate
		ptm, err := GetUnstructuredMapBySpecificPath(c.DevModeAction.PodTemplatePath, um.Object)
		if err != nil {
			return err
		}

		ptm["spec"] = &ps.Spec

		jsonObj, err := json.Marshal(um)
		if err != nil {
			return errors.WithStack(err)
		}

		infos, err := c.Client.GetResourceInfoFromString(string(jsonObj), true)
		if err != nil {
			return err
		}

		if len(infos) != 1 {
			return errors.New(fmt.Sprintf("ResourceInfo' num is %d(not 1?)", len(infos)))
		}

		log.Infof("Creating %s(%v)", infos[0].Name, infos[0].Object.GetObjectKind().GroupVersionKind())
		err = c.Client.ApplyResourceInfo(infos[0], nil)
		if err != nil {
			return err
		}

		gvk := infos[0].Object.GetObjectKind().GroupVersionKind()
		kind := gvk.Kind
		if gvk.Version != "" {
			kind += "." + gvk.Version
		}
		if gvk.Group != "" {
			kind += "." + gvk.Group
		}

		for _, item := range c.DevModeAction.ScalePatches {
			log.Infof("Patching %s", item.Patch)
			if err = c.Client.Patch(kind, infos[0].Name, item.Patch, item.Type); err != nil {
				return err
			}
		}

		c.patchAfterDevContainerReplaced(ops.Container, kind, infos[0].Name)
	} else {
		labelsMap, err := c.getDuplicateLabelsMap()
		if err != nil {
			return err
		}
		if ps, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}
		generatedDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   c.getDuplicateResourceName(),
				Labels: labelsMap,
			},
			Spec: appsv1.DeploymentSpec{
				Template: *ps,
			},
		}
		generatedDeployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
		generatedDeployment.Spec.Template.Labels = labelsMap
		generatedDeployment.ResourceVersion = ""
		generatedDeployment.Spec.Template.Spec.NodeName = ""

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(
				&generatedDeployment.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, true,
			)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(
			&generatedDeployment.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes,
		)

		// Create generated deployment
		if _, err = c.Client.CreateDeploymentAndWait(generatedDeployment); err != nil {
			return err
		}

		c.patchAfterDevContainerReplaced(ops.Container, generatedDeployment.Kind, generatedDeployment.Name)
	}

	return waitingPodToBeReady(c.GetDuplicateDevModePodName)
}

func (c *Controller) GetDuplicateModePodList() ([]v1.Pod, error) {
	labelsMap, err := c.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	return c.Client.Labels(labelsMap).ListPods()
}

func (c *Controller) GetDevModePodName() (string, error) {
	pods, err := c.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPodName(pods...)
}

func (c *Controller) GetDuplicateDevModePodName() (string, error) {
	pods, err := c.GetDuplicateModePodList()
	if err != nil {
		return "", err
	}
	return findDevPodName(pods...)
}

func (c *Controller) DuplicateModeRollBack() error {
	lmap, err := c.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	t := string(c.Type)
	if c.DevModeAction.Create {
		t = "deployment"
	}
	infos, err := c.Client.Labels(lmap).ListResourceInfo(t)
	if err != nil {
		return err
	}

	if len(infos) != 1 {
		return errors.New(fmt.Sprintf("%d duplicate %s found?", len(infos), t))
	}

	return clientgoutils.DeleteResourceInfo(infos[0])
}
