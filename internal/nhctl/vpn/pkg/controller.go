package pkg

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

type Scalable interface {
	ScaleToZero() (map[string]string, []v1.ContainerPort, string, error)
	Cancel() error
	Reset() error
}

func ToInboundPodName(resourceType, resourceName string) string {
	return fmt.Sprintf("%s-%s-shadow", resourceType, resourceName)
}
