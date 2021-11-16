package pkg

import (
	v1 "k8s.io/api/core/v1"
)

type Scalable interface {
	ScaleToZero() (map[string]string, []v1.ContainerPort, error)
	Cancel() error
}
