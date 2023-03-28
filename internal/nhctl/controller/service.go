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

func (c *Controller) CheckIfExist() error {
	_, err := c.GetUnstructured()
	return err
}

func (c *Controller) GetOriginalContainers() ([]v1.Container, error) {
	return GetOriginalContainers(c.Client, c.Type, c.Name, c.DevModeAction.PodTemplatePath)
}

func GetOriginalContainers(client *clientgoutils.ClientGoUtils, workloadType base.SvcType, workloadName, path string) ([]v1.Container, error) {
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

func (c *Controller) getDuplicateLabelsMap() map[string]string {

	labelsMap := map[string]string{
		IdentifierKey:             c.Identifier,
		OriginWorkloadNameKey:     c.Name,
		OriginWorkloadTypeKey:     string(c.Type),
		_const.DevWorkloadIgnored: "true",
	}
	return labelsMap
}

func (c *Controller) getDevContainerAnnotations(devContainer string, originAnnos map[string]string) map[string]string {
	if len(originAnnos) == 0 {
		originAnnos = map[string]string{}
	}

	originAnnos[_const.NocalhostDevContainerAnnotations] = c.GetDevContainerName(devContainer)
	return originAnnos
}

func (c *Controller) getDuplicateResourceName() string {
	uuid, _ := utils.GetShortUuid()
	return strings.Join([]string{c.Name, c.Identifier[0:5], uuid}, "-")
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

func (c *Controller) getGeneratedDeploymentName() string {
	id, _ := utils.GetShortUuid()
	return fmt.Sprintf("%s-gen-%s", c.Name, id)
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

	// Check if already in DevMode
	podTemplate, err := GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, unstructuredObj.Object)
	if err != nil {
		return err
	}

	for _, container := range podTemplate.Spec.Containers {
		if container.Name == _const.NocalhostDefaultDevSidecarName {
			return errors.New(fmt.Sprintf("Container %s already exists, you need to reset it first", container.Name))
		}
	}

	mBytes, _ := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{_const.OriginWorkloadDefinition: string(originalSpecJson)}}})

	if err = c.Client.Patch(c.Type.String(), c.Name, string(mBytes), "merge"); err != nil {
		return err
	}
	log.Info("Original manifest recorded")

	log.Info("Executing ScalePatches...")
	for _, item := range c.DevModeAction.ScalePatches {
		log.Infof("Patching %s(%s)", item.Patch, item.Type)
		if err := c.Client.Patch(c.Type.String(), c.Name, item.Patch, item.Type); err != nil {
			return err
		}
	}

	podSpec := &podTemplate.Spec

	devContainer, sideCarContainer, devModeVolumes, err :=
		c.genContainersAndVolumes(podSpec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(podSpec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	if !c.DevModeAction.Create {

		log.Info("Update strategy to RECREATE")
		specPath := "/spec/strategy"
		strategy := &appsv1.DeploymentStrategy{
			Type:          appsv1.RecreateDeploymentStrategyType,
			RollingUpdate: nil,
		}
		jsonPatches := make([]jsonPatch, 0)
		jsonPatches = append(
			jsonPatches, jsonPatch{
				Op:    "replace",
				Path:  specPath,
				Value: strategy,
			},
		)
		bys, _ := json.Marshal(jsonPatches)
		if err = c.Client.Patch(c.Type.String(), c.Name, string(bys), "json"); err != nil {
			log.WarnE(err, "")
		}

		log.Info("Patching development container...")
		specPath = c.DevModeAction.PodTemplatePath + "/spec"
		jsonPatches = make([]jsonPatch, 0)
		jsonPatches = append(
			jsonPatches, jsonPatch{
				Op:    "replace",
				Path:  specPath,
				Value: podSpec,
			},
		)
		bys, _ = json.Marshal(jsonPatches)

		if err = c.Client.Patch(c.Type.String(), c.Name, string(bys), "json"); err != nil {
			return err
		}

		log.Info("Patching development annotations...")
		specPath = c.DevModeAction.PodTemplatePath + "/metadata/annotations"
		jsonPatches = make([]jsonPatch, 0)
		jsonPatches = append(
			jsonPatches, jsonPatch{
				Op:    "replace",
				Path:  specPath,
				Value: c.getDevContainerAnnotations(ops.Container, podTemplate.GetAnnotations()),
			},
		)
		bys, _ = json.Marshal(jsonPatches)
		patchContent := string(bys)

		if err = c.Client.Patch(c.Type.String(), c.Name, patchContent, "json"); err != nil {
			return err
		}

		c.patchAfterDevContainerReplaced(ops.Container, c.Type.String(), c.Name)
	} else {
		// Some workload's pod may not have labels, such as cronjob, we need to give it one
		if len(podTemplate.Labels) == 0 {
			podTemplate.Labels = c.getGeneratedDeploymentLabels()
		}

		podTemplate.Annotations = c.getDevContainerAnnotations(ops.Container, podTemplate.Annotations)
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
		generatedDeployment.Spec.Strategy.RollingUpdate = nil
		generatedDeployment.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		if _, err = c.Client.CreateDeployment(generatedDeployment); err != nil {
			return err
		}
		c.patchAfterDevContainerReplaced(ops.Container, "deployment", generatedDeployment.Name)
	}

	delete(podTemplate.Labels, "pod-template-hash")
	c.devModePodLabels = podTemplate.Labels

	c.waitDevPodToBeReady()
	return nil
}

func (c *Controller) waitDevPodToBeReady() {
	gvr := c.Client.ResourceFor("pod", false)
	gvk, gvkErr := c.Client.KindFor(gvr)
	if gvkErr != nil {
		log.Infof(
			"(Can Be Ignore, does not affect the actual function) "+
				"Error to find workload's GVK "+
				"(Event Watcher when dev start will be invalid), Error:%s", gvk,
		)
	}

	log.Infof("Now waiting dev mode to start...\n")

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

	readyChan := make(chan struct{}, 0)
	stopChan := make(chan struct{}, 0)
	defer close(stopChan)

	watcher.NewSimpleWatcher(
		c.Client,
		"pod",
		metav1.ListOptions{
			LabelSelector: c.devModePodLabels.AsSelector().String(),
		},
		stopChan,
		func(key string, object interface{}, quitChan <-chan struct{}) {
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
						readyChan <- struct{}{}
					}
					return
				}
			}
		},
		nil,
	)

	if gvkErr == nil && !gvk.Empty() {
		apiVersion, kind := gvk.ToAPIVersionAndKind()

		temp := make(chan struct{})
		defer close(temp)
		watcher.NewSimpleWatcher(
			c.Client,
			"event",
			metav1.ListOptions{},
			temp,
			func(key string, object interface{}, quitChan <-chan struct{}) {
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

						printer.ChangeContent(
							fmt.Sprintf(
								"### Notable events: %s: %s ###", event.Reason, event.Message,
							),
						)
						return
					}
				}
			},
			nil,
		)
	}

	select {
	case _, _ = <-stopChan:
	case <-readyChan:
	}
}

func (c *Controller) CheckDevModePodIsRunning() (string, error) {
	pods, err := c.Client.Labels(c.devModePodLabels).ListPods()
	if err != nil {
		return "", err
	}
	return findDevPodName(pods...)
}

func (c *Controller) GetDuplicateModePodList() ([]v1.Pod, error) {
	return c.Client.Labels(c.getDuplicateLabelsMap()).ListPods()
}

func (c *Controller) GetDevModePodName() (string, error) {
	pods, err := c.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPodName(pods...)
}

func (c *Controller) DuplicateModeRollBack() error {
	lmap := c.getDuplicateLabelsMap()
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

func (c *Controller) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	if c.DevModeType.IsDuplicateDevMode() {
		return c.ReplaceDuplicateModeImage(ctx, ops)
	}
	return c.PatchDevModeManifest(ctx, ops)
}

func (c *Controller) RollBack(reset bool) error {
	if c.DevModeType.IsDuplicateDevMode() {
		return c.DuplicateModeRollBack()
	}
	return c.RollbackFromAnnotation(reset)
}

// GetPVCsBySvc
// Get all PersistVolumeClaims created by specified service
func (c *Controller) GetPVCsBySvc() ([]v1.PersistentVolumeClaim, error) {
	return c.Client.GetPvcByLabels(
		map[string]string{_const.AppLabel: c.AppName, _const.ServiceLabel: c.Name,
			_const.ServiceTypeLabel: string(c.Type)},
	)
}

func GetDefaultPodName(ctx context.Context, p *Controller) (string, error) {
	var (
		podList []v1.Pod
		err     error
	)
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", p.Name))
		default:
			podList, err = p.GetPodList()
		}
		if err != nil || len(podList) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", p.Name)
			time.Sleep(time.Second)
		} else {
			return podList[0].Name, nil
		}
	}
}
