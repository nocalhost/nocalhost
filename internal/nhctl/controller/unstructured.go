/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

const (
	OriginSpecJson = "nocalhost.origin.spec.json" // deprecated
)

func (c *Controller) GetUnstructured() (unstructuredMap *unstructured.Unstructured, err error) {
	return c.Client.GetUnstructured(string(c.Type), c.Name)
}

func (c *Controller) GetPodTemplate() (*corev1.PodTemplateSpec, error) {
	um, err := c.GetUnstructured()
	if err != nil {
		return nil, err
	}
	return GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object)
}

// GetPodList
// If in Replace DevMode and DevModeAction.Create is true, return pods of generated deployment
// Others, return pods of the workload
func (c *Controller) GetPodList() ([]corev1.Pod, error) {
	if c.IsInReplaceDevMode() && c.DevModeAction.Create {
		return c.ListPodOfGeneratedDeployment()
	}
	pt, err := c.GetPodTemplate()
	if err != nil {
		return nil, err
	}
	delete(pt.Labels, "pod-template-hash")
	return c.Client.Labels(pt.Labels).ListPods()
}

func (c *Controller) getGeneratedDeployment() (*v1.Deployment, error) {
	ds, err := c.Client.Labels(c.getGeneratedDeploymentLabels()).ListDeployments()
	if err != nil {
		return nil, err
	}
	if len(ds) != 1 {
		return nil, errors.New(fmt.Sprintf("Generated deployment is %d(not 1?)", len(ds)))
	}
	return &ds[0], nil
}

func (c *Controller) ListPodOfGeneratedDeployment() ([]corev1.Pod, error) {
	ds, err := c.getGeneratedDeployment()
	if err != nil {
		return nil, err
	}
	return c.Client.ListPodsOfDeployment(ds.Name)
}

func (c *Controller) IncreaseDevModeCount() error {
	um, err := c.GetUnstructured()
	if err != nil {
		return err
	}
	count := 0
	devModeCount, err := GetAnnotationFromUnstructured(um, _const.DevModeCount)
	if err != nil {
		count = 1
	} else {
		count, _ = strconv.Atoi(devModeCount)
		count++
	}

	// strategic patch don't work in CRD
	//m := map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{_const.DevModeCount: fmt.Sprintf("%d", count)}}}
	//mBytes, _ := json.Marshal(m)
	//return c.Client.Patch(c.Type.String(), c.Name, string(mBytes), "strategic")

	specPath := "/metadata/annotations/" + strings.ReplaceAll(_const.DevModeCount, "/", "~1")
	jsonPatches := make([]jsonPatch, 0)
	jsonPatches = append(
		jsonPatches, jsonPatch{
			Op:    "add",
			Path:  specPath,
			Value: fmt.Sprintf("%d", count),
		},
	)
	bys, _ := json.Marshal(jsonPatches)

	return c.Client.Patch(c.Type.String(), c.Name, string(bys), "json")
}

func (c *Controller) DecreaseDevModeCount() error {
	um, err := c.GetUnstructured()
	if err != nil {
		return err
	}
	devModeCount, err := GetAnnotationFromUnstructured(um, _const.DevModeCount)
	if err != nil {
		return nil
	}
	count, _ := strconv.Atoi(devModeCount)
	if count > 0 {
		count--
	}

	// https://stackoverflow.com/questions/55573724/create-a-patch-to-add-a-kubernetes-annotation/60402927
	specPath := "/metadata/annotations/" + strings.ReplaceAll(_const.DevModeCount, "/", "~1")
	jsonPatches := make([]jsonPatch, 0)
	jsonPatches = append(
		jsonPatches, jsonPatch{
			Op:    "add",
			Path:  specPath,
			Value: fmt.Sprintf("%d", count),
		},
	)
	bys, _ := json.Marshal(jsonPatches)

	return c.Client.Patch(c.Type.String(), c.Name, string(bys), "json")
}

func (c *Controller) RollbackFromAnnotation() error {

	if c.DevModeAction.Create {
		log.Info("Destroying generated deployment")
		ds, err := c.getGeneratedDeployment()
		if err != nil {
			return err
		}
		if err = c.Client.DeleteDeployment(ds.Name, false); err != nil {
			return err
		}
	}

	devModeWorkload, err := c.GetUnstructured()
	if err != nil {
		return err
	}

	osj, err := GetAnnotationFromUnstructured(devModeWorkload, _const.OriginWorkloadDefinition)
	if err != nil {
		return err
	}
	log.Infof("Annotation %s found, use it", _const.OriginWorkloadDefinition)

	originalWorkload, err := c.Client.GetResourceInfoFromString(osj, true)
	if err != nil {
		return err
	}

	if len(originalWorkload) != 1 {
		return errors.New(fmt.Sprintf("Original workload is not 1(%d)?", len(originalWorkload)))
	}

	// Recreate
	if err := clientgoutils.DeleteResourceInfo(originalWorkload[0]); err != nil {
		return err
	}

	originalWorkload, err = c.Client.GetResourceInfoFromString(osj, true)
	if err != nil {
		return err
	}

	if len(originalWorkload) != 1 {
		return errors.New(fmt.Sprintf("Original workload is not 1(%d)?", len(originalWorkload)))
	}

	return c.Client.ApplyResourceInfo(originalWorkload[0], nil)

	//originalWorkload, err := c.Client.GetUnstructuredFromString(osj)
	//if err != nil {
	//	return err
	//}

	//specMap, ok := originalWorkload.Object["spec"]
	//if !ok {
	//	return errors.New("Spec not found in workload definition")
	//}
	//
	//jsonPatches := make([]jsonPatch, 0)
	//jsonPatches = append(jsonPatches, jsonPatch{
	//	Op:    "replace",
	//	Path:  "/spec",
	//	Value: specMap,
	//})
	//
	//bys, _ := json.Marshal(jsonPatches)
	//return c.Client.Patch(c.Type.String(), c.Name, string(bys), "json")
}

// GetUnstructuredMapBySpecificPath Path must be like: /spec/template
func GetUnstructuredMapBySpecificPath(path string, u map[string]interface{}) (map[string]interface{}, error) {
	if strings.HasPrefix(path, "\\/") {
		return nil, errors.New(fmt.Sprintf("Path %s invalid. It must be like: /spec/template, and start with /", path))
	}
	path = strings.TrimPrefix(path, "/")
	pathItems := strings.Split(path, "/")
	arrayIndex := -1
	splitIndex := -1
	for i, item := range pathItems {
		if item == "" {
			continue
		}
		index, err := strconv.Atoi(item)
		if err == nil { // There is a slice
			splitIndex = i
			arrayIndex = index
			break
		}
	}

	if splitIndex == -1 {
		result, ok, err := unstructured.NestedMap(u, pathItems...)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !ok {
			return nil, errors.New(fmt.Sprintf("PodTemplate path %s not found", path))
		}
		return result, nil
	}

	mapSlice, ok, err := unstructured.NestedSlice(u, pathItems[0:splitIndex]...)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find nested slice in unstructured map")
	}
	if !ok {
		return nil, errors.New(fmt.Sprintf("PodTemplate path %s not found", path))
	}

	if len(mapSlice) < arrayIndex+1 {
		return nil, errors.New(fmt.Sprintf("Array index %d in path %s is out of boundary", arrayIndex, path))
	}

	p, ok := mapSlice[arrayIndex].(map[string]interface{})
	if !ok {
		return nil, errors.New(fmt.Sprintf("Slice item in path %s is not a unstructured map?", path))
	}

	secondMap, ok, err := unstructured.NestedMap(p, pathItems[splitIndex+1:]...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !ok {
		return nil, errors.New(fmt.Sprintf("PodTemplate path %s not found", path))
	}
	return secondMap, nil
}

func GetPodTemplateFromSpecPath(path string, unstructuredObj map[string]interface{}) (*corev1.PodTemplateSpec, error) {
	currentPathMap, err := GetUnstructuredMapBySpecificPath(path, unstructuredObj)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(currentPathMap)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	p := &corev1.PodTemplateSpec{}
	if err = json.Unmarshal(jsonBytes, p); err != nil {
		return nil, errors.WithStack(err)
	} else {
		return p, nil
	}
}

// GetAnnotationFromUnstructured get annotation from unstructured
func GetAnnotationFromUnstructured(u *unstructured.Unstructured, key string) (string, error) {
	as := u.GetAnnotations()
	if len(as) == 0 {
		return "", errors.New("Annotations is nil")
	}

	if v, ok := as[key]; !ok {
		return "", errors.New(fmt.Sprintf("Annotation %s not found", key))
	} else {
		return v, nil
	}
}

func RemoveUselessInfo(u *unstructured.Unstructured) {
	if u == nil {
		return
	}

	delete(u.Object, "status")

	u.SetManagedFields(nil)
	u.SetResourceVersion("")
	u.SetCreationTimestamp(metav1.NewTime(time.Time{}))
	u.SetManagedFields(nil)
	u.SetUID("")
	u.SetGeneration(0)
	a := u.GetAnnotations()
	if len(a) == 0 {
		return
	}
	delete(a, _const.OriginWorkloadDefinition)
	delete(a, "kubectl.kubernetes.io/last-applied-configuration")
	delete(a, OriginSpecJson) // remove deprecated annotation
	u.SetAnnotations(a)
}

func AddItemToUnstructuredMap(path string, u map[string]interface{}, key string, item map[string]interface{}) error {
	ps := strings.Split(path, "/")
	currentMap := u
	for _, p := range ps {
		if p == "" {
			continue
		}
		tM, ok := currentMap[p]
		if !ok {
			return errors.New(fmt.Sprintf("Add item to UnstructuredMap failed in %s", p))
		}

		tm, ok := tM.(map[string]interface{})
		if !ok {
			return errors.New(fmt.Sprintf("Add item to UnstructuredMap failed in %s", p))
		}
		currentMap = tm
	}
	currentMap[key] = item
	return nil
}

func (c *Controller) PatchDuplicateInfo(u map[string]interface{}) error {
	labelsMap, err := c.getDuplicateLabelsMap()
	if err != nil {
		return err
	}

	metaM, ok := u["metadata"]
	if !ok {
		return errors.New("metadata not found")
	}

	mm, ok := metaM.(map[string]interface{})
	if !ok {
		return errors.New("metadata invalid")
	}

	//dep.Name = d.getDuplicateResourceName()
	mm["name"] = c.getDuplicateResourceName()
	mm["labels"] = labelsMap

	delete(mm, "resourceVersion") //dep.ResourceVersion = ""
	delete(u, "status")           //dep.Status = appsv1.DeploymentStatus{}

	// todo
	//dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	pathItems := strings.Split(c.DevModeAction.PodTemplatePath, "/")
	path := strings.Join(pathItems[:len(pathItems)-1], "/")
	lm := map[string]interface{}{"matchLabels": labelsMap}
	err = AddItemToUnstructuredMap(path, u, "selector", lm)
	if err != nil {
		return err
	}

	li := map[string]interface{}{}
	for s, s2 := range labelsMap {
		li[s] = s2
	}

	//dep.Spec.Template.Labels = labelsMap
	err = AddItemToUnstructuredMap(c.DevModeAction.PodTemplatePath+"/"+"metadata", u, "labels", li)
	if err != nil {
		return err
	}
	return nil
}
