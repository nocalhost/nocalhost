/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

type PatchItem struct {
	Patch string `json:"patch" yaml:"patch"`
	Type  string `json:"type,omitempty" yaml:"type,omitempty"`
}
