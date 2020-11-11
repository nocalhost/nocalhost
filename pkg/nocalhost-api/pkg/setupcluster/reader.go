package setupcluster

import corev1 "k8s.io/api/core/v1"

func GetServiceAccountSecretByKey(c *corev1.Secret, key string) string {
	return string(c.Data[key])
}
