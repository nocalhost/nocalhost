package sleep

import (
	"encoding/json"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func Update(c *clientgo.GoClient, id uint64, ns string, conf model.SleepConfig) error {
	// 1. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KSleepConfig: ternary(len(conf.ByWeek) == 0, "", stringify(conf)).(string),
				KForceAsleep: "",
				KForceWakeup: "",
			},
		},
	})
	_, err := c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// 2. write to database
	err = service.Svc.ClusterUser().Modify(context.TODO(), id, map[string]interface{}{
		"sleep_config": &conf,
	})
	if err != nil {
		return err
	}
	return nil
}
