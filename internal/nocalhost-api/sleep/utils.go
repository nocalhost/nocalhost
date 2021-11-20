package sleep

import (
	"context"
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func CreateSleepConfig(c *clientgo.GoClient, ns string, config string) error {
	marshal, _ := json.Marshal(map[string]map[string]map[string]string{
		"metadata": {
			"annotations": {
				"nocalhost.dev.sleep/config": config,
			},
		},
	})

	_, err := c.
		Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, marshal, metav1.PatchOptions{})
	return err
}

func DeleteSleepConfig(c *clientgo.GoClient, ns string) error {
	namespace, err := c.
		Clientset().
		CoreV1().
		Namespaces().
		Get(context.TODO(), ns, metav1.GetOptions{})

	if err != nil {
		return err
	}

	delete(namespace.Annotations, "nocalhost.dev.sleep/config")
	delete(namespace.Annotations, "nocalhost.dev.sleep/status")
	delete(namespace.Annotations, "nocalhost.dev.sleep/force-sleep")
	delete(namespace.Annotations, "nocalhost.dev.sleep/force-wakeup")
	delete(namespace.Annotations, "nocalhost.dev.sleep/origin-replicas")

	_, err = c.
		Clientset().
		CoreV1().
		Namespaces().
		Update(context.TODO(), namespace, metav1.UpdateOptions{})
	return err
}
