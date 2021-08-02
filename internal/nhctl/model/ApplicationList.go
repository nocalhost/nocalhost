/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package model

type Namespace struct {
	Namespace   string             `json:"namespace" yaml:"namespace"`
	Application []*ApplicationInfo `json:"application" yaml:"application"`
}

type ApplicationInfo struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}
