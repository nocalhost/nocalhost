package clientgoutils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) CreateSecret(secret *corev1.Secret) (*corev1.Secret, error) {
	return c.GetSecretClient().Create(c.ctx, secret, metav1.CreateOptions{})
}

func (c *ClientGoUtils) UpdateSecret(secret *corev1.Secret) (*corev1.Secret, error) {
	return c.GetSecretClient().Update(c.ctx, secret, metav1.UpdateOptions{})
}

func (c *ClientGoUtils) GetSecret(name string) (*corev1.Secret, error) {
	return c.GetSecretClient().Get(c.ctx, name, metav1.GetOptions{})
}

func (c *ClientGoUtils) DeleteSecret(name string) error {
	return c.GetSecretClient().Delete(c.ctx, name, metav1.DeleteOptions{})
}
