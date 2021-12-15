package sleep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"time"
)

func Wakeup(c *clientgo.GoClient, s *model.ClusterUserModel, force bool) error {
	// 1. check namespace
	namespace, err := c.Clientset().
		CoreV1().
		Namespaces().
		Get(context.TODO(), s.Namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 2. check replicas
	if namespace.Annotations == nil || len(namespace.Annotations[KReplicas]) == 0 {
		return errors.New(fmt.Sprintf("cannot find `%s` from annotations", KReplicas))
	}
	var replicas map[string]int32
	err = json.Unmarshal([]byte(namespace.Annotations[KReplicas]), &replicas)
	if err != nil {
		return err
	}

	// 3. restore Deployment
	deps, err := c.Clientset().
		AppsV1().
		Deployments(s.Namespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dp := range deps.Items {
		if ignorable(dp.Annotations) {
			continue
		}
		n, ok := replicas[KDeployment+":"+dp.Name]
		if ok {
			dp.Spec.Replicas = &n
			_, err = c.Clientset().
				AppsV1().
				Deployments(s.Namespace).
				Update(context.TODO(), &dp, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 4. restore StatefulSet
	sets, err := c.Clientset().
		AppsV1().
		StatefulSets(s.Namespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, st := range sets.Items {
		if ignorable(st.Annotations) {
			continue
		}
		n, ok := replicas[KStatefulSet+":"+st.Name]
		if ok {
			st.Spec.Replicas = &n
			_, err = c.Clientset().
				AppsV1().
				StatefulSets(s.Namespace).
				Update(context.TODO(), &st, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 5. restore CronJob
	jobs, err := c.Clientset().
		BatchV1beta1().
		CronJobs(s.Namespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, jb := range jobs.Items {
		if ignorable(jb.Annotations) {
			continue
		}
		jb.Spec.Suspend = &falsely
		_, err = c.Clientset().
			BatchV1beta1().
			CronJobs(s.Namespace).
			Update(context.TODO(), &jb, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// 6. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KSleepStatus: KWakeup,
				KForceAsleep: "",
				KForceWakeup: ternary(force, timestamp(), "").(string),
			},
		},
	})
	_, err = c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), s.Namespace, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// 7. write to database
	diff := 0
	if s.SleepAt != nil {
		diff = int(time.Now().Sub(*s.SleepAt) / time.Minute)
	}
	return cluster_user.
		NewClusterUserService().
		Modify(context.TODO(), s.ID, map[string]interface{}{
			"sleep_at":     nil,
			"sleep_status": KWakeup,
			"sleep_minute": gorm.Expr("`sleep_minute` + ?", diff),
		})
}
