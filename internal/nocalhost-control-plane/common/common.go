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
