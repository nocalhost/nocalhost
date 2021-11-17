package pkg

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

type Scalable interface {
	ScaleToZero() (map[string]string, []v1.ContainerPort, error)
	Cancel() error
	Reset() error
}

func toInboundPodName(resourceType, resourceName string) string {
	return fmt.Sprintf("%s-%s-shadow", resourceType, resourceName)
}
