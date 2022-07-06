/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package common

import "nocalhost/internal/nhctl/controller"

const (
	ListenerAddress    = "0.0.0.0"
	PassthroughCluster = "PassthroughCluster"
)

const (
	MeshKey        = controller.AnnotationMeshEnable
	MeshUUIDKEY    = controller.AnnotationMeshUuid
	MeshTypeKEY    = controller.AnnotationMeshType
	MeshHeaderKey  = controller.AnnotationMeshHeaderKey
	MeshHeaderVal  = controller.AnnotationMeshHeaderValue
	MeshDevType    = controller.AnnotationMeshTypeDev
	MeshOriginType = controller.AnnotationMeshTypeOrigin
)
