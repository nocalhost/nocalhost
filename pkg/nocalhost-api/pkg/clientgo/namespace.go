package clientgo

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (c *GoClient) GetNamespace(namespace string) (*corev1.Namespace, error) {
	resources, err := c.client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	return resources, errors.WithStack(err)
}

func (c *GoClient) GetClientSet() *kubernetes.Clientset {
	return c.client
}
