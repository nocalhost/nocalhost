/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

func (c *Controller) GetUnstructuredMap() (unstructuredMap map[string]interface{}, err error) {
	return c.Client.GetUnstructuredMap(string(c.Type), c.Name)
}

func (c *Controller) GetPodTemplate() (*corev1.PodTemplateSpec, error) {
	um, err := c.GetUnstructuredMap()
	if err != nil {
		return nil, err
	}
	return GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um)
}

func (c *Controller) GetPodList() ([]corev1.Pod, error) {
	pt, err := c.GetPodTemplate()
	if err != nil {
		return nil, err
	}
	delete(pt.Labels, "pod-template-hash")
	return c.Client.Labels(pt.Labels).ListPods()
}

func (c *Controller) RollbackFromAnnotation() error {
	unstructuredMap, err := c.GetUnstructuredMap()
	if err != nil {
		return err
	}

	osj, err := GetAnnotationFromUnstructuredMap(unstructuredMap, _const.OriginWorkloadDefinition)
	if err != nil {
		return err
	}
	log.Infof("Annotation %s found, use it", _const.OriginWorkloadDefinition)

	originUnstructuredMap, err := c.Client.GetUnstructuredMapFromString(osj)
	if err != nil {
		return err
	}

	specMap, ok := originUnstructuredMap["spec"]
	if !ok {
		return errors.New("spec not found in annotation")
	}

	jsonPatches := make([]jsonPatch, 0)
	jsonPatches = append(jsonPatches, jsonPatch{
		Op:    "replace",
		Path:  "/spec",
		Value: specMap,
	})

	bys, _ := json.Marshal(jsonPatches)
	return c.Client.Patch(c.Type.String(), c.Name, string(bys), "json")
}

func GetPodTemplateFromSpecPath(path string, unstructuredObj map[string]interface{}) (*corev1.PodTemplateSpec, error) {
	pathItems := strings.Split(path, "/")
	currentPathMap := unstructuredObj
	for _, item := range pathItems {
		if item != "" {
			if m, ok := currentPathMap[item]; ok {
				currentPathMap = m.(map[string]interface{})
			} else {
				return nil, errors.New(fmt.Sprintf("Invalid path: %s", item))
			}
		}
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

// GetAnnotationFromUnstructuredMap todo: check map if nil
func GetAnnotationFromUnstructuredMap(u map[string]interface{}, key string) (string, error) {
	meta, ok := u["metadata"]
	if !ok {
		return "", errors.New("origin spec json not fount(metadata)")
	}
	metaMap := meta.(map[string]interface{})
	annotations, ok := metaMap["annotations"]
	if !ok {
		return "", errors.New("origin spec json not fount(annotations)")
	}
	annotationsMap := annotations.(map[string]interface{})
	originJson, ok := annotationsMap[key]
	if !ok {
		return "", errors.New("origin spec json not fount")
	}
	return originJson.(string), nil
}
