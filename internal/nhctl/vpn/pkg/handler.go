/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	v1 "k8s.io/api/core/v1"
)

type Scalable interface {
	ScaleToZero() (labels, annotations map[string]string, ports []v1.ContainerPort, backupStr string, err error)
	ToInboundPodName() string
	Reset() error
}
