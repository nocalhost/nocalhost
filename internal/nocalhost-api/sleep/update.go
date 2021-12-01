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

func Update(c *clientgo.GoClient, id uint64, ns string, conf model.SleepConfig) (*model.ClusterUserModel, error) {
	// 1. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KConfig:      Ternary(len(conf.ByWeek) == 0, "" , Stringify(conf)).(string),
				KStatus:      "",
				KForceSleep:  "",
				KForceWakeup: "",
			},
		},
	})
	_, err := c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})

	// 2. write to database
	result, err := service.Svc.ClusterUser().Update(context.TODO(), &model.ClusterUserModel{
		ID: id,
		SleepConfig: &conf,
		SleepSaving: Calc(&conf.ByWeek),
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
