/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"nocalhost/internal/nhctl/common/base"
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
	if c.DevModeType.IsDuplicateDevMode() {
		return c.GetDuplicateModePodList()
	}
	if c.Type == base.Pod {
		pod, err := c.Client.GetPod(c.Name)
		if err != nil {
			return nil, err
		}
		return []corev1.Pod{*pod}, nil
	}
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

func (c *Controller) getGeneratedDeployment() ([]v1.Deployment, error) {
	ds, err := c.Client.Labels(c.getGeneratedDeploymentLabels()).ListDeployments()
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func (c *Controller) ListPodOfGeneratedDeployment() ([]corev1.Pod, error) {
	ds, err := c.getGeneratedDeployment()
	if err != nil {
		return nil, err
	}
	if len(ds) != 1 {
		return nil, errors.New(fmt.Sprintf("Generated deployment is %d(not 1?)", len(ds)))
	}
	return c.Client.ListPodsOfDeployment(ds[0].Name)
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

func (c *Controller) RollbackFromAnnotation(reset bool) error {

	if c.DevModeAction.Create {
		log.Info("Destroying generated deployment")
		ds, err := c.getGeneratedDeployment()
		if err != nil {
			return err
		}
		if len(ds) != 1 {
			if reset {
				for _, d := range ds {
					// Clean up generated deployment
					if err = c.Client.DeleteDeployment(d.Name, false); err != nil {
						log.WarnE(err, "")
					}
				}
			} else {
				return errors.New(fmt.Sprintf("Generated deployment is %d(not 1?)", len(ds)))
			}
		} else {
			if err = c.Client.DeleteDeployment(ds[0].Name, false); err != nil {
				return err
			}
		}
	}

	devModeWorkload, err := c.GetUnstructured()
	if err != nil {
		return err
	}

	var originalWorkload []*resource.Info
	osj, err := GetAnnotationFromUnstructured(devModeWorkload, _const.OriginWorkloadDefinition)
	if err != nil {
		// For compatibility
		log.Infof("Annotation %s not found, finding %s", _const.OriginWorkloadDefinition, OriginSpecJson)
		osj2, err := GetAnnotationFromUnstructured(devModeWorkload, OriginSpecJson)
		if err != nil {
			return err
		}

		osj2 = strings.Trim(osj2, "\"")

		mj, err := devModeWorkload.MarshalJSON()
		if err != nil {
			return err
		}
		osj = string(mj)
		if osj, err = sjson.SetRaw(osj, "spec", osj2); err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Infof("Annotation %s found, use it", _const.OriginWorkloadDefinition)
	}

	originalWorkload, err = c.Client.GetResourceInfoFromString(osj, true)
	if err != nil {
		return err
	}

	if len(originalWorkload) != 1 {
		return errors.New(fmt.Sprintf("Original workload is not 1(%d)?", len(originalWorkload)))
	}

	if !c.DevModeAction.Create {
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

		if um, ok := originalWorkload[0].Object.(*unstructured.Unstructured); ok {
			ans := um.GetAnnotations()
			if ans == nil {
				ans = map[string]string{}
			}
			ans["nocalhost-dep-ignore"] = "true"
			ans[_const.NocalhostApplicationName] = c.AppName
			ans[_const.NocalhostApplicationNamespace] = c.NameSpace
			um.SetAnnotations(ans)
		}
	}

	return c.Client.ApplyResourceInfo(originalWorkload[0], nil)
}

// GetUnstructuredMapByPath Path must be like: /spec/template
func GetUnstructuredMapByPath(path string, u map[string]interface{}) (map[string]interface{}, error) {
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

	p := &corev1.PodTemplateSpec{}

	// For Pod
	if path == "" {
		podSpecMap, err := GetUnstructuredMapByPath("/spec", unstructuredObj)
		if err != nil {
			return nil, err
		}
		podBytes, err := json.Marshal(podSpecMap)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return p, errors.WithStack(json.Unmarshal(podBytes, &p.Spec))
	}

	currentPathMap, err := GetUnstructuredMapByPath(path, unstructuredObj)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(currentPathMap)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p, errors.WithStack(json.Unmarshal(jsonBytes, p))
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
