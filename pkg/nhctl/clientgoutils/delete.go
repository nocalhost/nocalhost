package clientgoutils

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) DeleteConfigMapByName(name string, namespace string) error {
	var err error
	if namespace == "" {
		namespace, err = c.GetDefaultNamespace()
		if err != nil {
			return err
		}
	}
	return c.ClientSet.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}
