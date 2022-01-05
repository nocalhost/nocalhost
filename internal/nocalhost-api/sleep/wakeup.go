package sleep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func Wakeup(c *clientgo.GoClient, s *model.ClusterUserModel, force bool) error {
	// 1. wakeup
	if err := wakeup(c.Clientset(), s.Namespace, force); err != nil {
		return err
	}

	// 2. write to database
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

func wakeup(c kubernetes.Interface, namespace string, force bool) error {
	// 1. check ns
	ns, err := c.CoreV1().
		Namespaces().
		Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 2. check replicas
	if ns.Annotations == nil || len(ns.Annotations[KReplicas]) == 0 {
		return errors.New(fmt.Sprintf("cannot find `%s` from annotations", KReplicas))
	}
	var replicas map[string]int32
	err = json.Unmarshal([]byte(ns.Annotations[KReplicas]), &replicas)
	if err != nil {
		return err
	}

	// 3. restore Deployment
	deps, err := c.AppsV1().
		Deployments(namespace).
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
			_, err = c.AppsV1().
				Deployments(namespace).
				Update(context.TODO(), &dp, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 4. restore StatefulSet
	sets, err := c.AppsV1().
		StatefulSets(namespace).
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
			if st.GetLabels()["app"] == "vcluster" {
				// TODO wakeup vcluster
				continue
			}
			st.Spec.Replicas = &n
			_, err = c.AppsV1().
				StatefulSets(namespace).
				Update(context.TODO(), &st, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 5. restore CronJob
	jobs, err := c.BatchV1beta1().
		CronJobs(namespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, jb := range jobs.Items {
		if ignorable(jb.Annotations) {
			continue
		}
		jb.Spec.Suspend = &falsely
		_, err = c.BatchV1beta1().
			CronJobs(namespace).
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
	_, err = c.CoreV1().
		Namespaces().
		Patch(context.TODO(), namespace, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}
