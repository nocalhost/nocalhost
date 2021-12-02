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
	"strings"
)

func (c *Controller) GetUnstructuredMap() (unstructuredMap map[string]interface{}, err error) {
	return c.Client.GetUnstructuredMap(string(c.Type), c.Name)
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
