/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package handler

import v1 "k8s.io/api/core/v1"

type Handler interface {
	InjectVPNContainer() error
	GetPod() ([]v1.Pod, error)
	Rollback(reset bool) error
}
