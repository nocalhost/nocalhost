package clientgoutils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"testing"
	"time"
)

func TestClientGoUtils_GetResourcesByRestClient(t *testing.T) {
	client, err := NewClientGoUtils("", time.Minute)
	Must(err)
	result := &corev1.PodList{}
	Must(client.GetResourcesByRestClient(&corev1.SchemeGroupVersion, "", ResourcePods, result))
	for _, item := range result.Items {
		fmt.Println(item.Name)
	}
}
