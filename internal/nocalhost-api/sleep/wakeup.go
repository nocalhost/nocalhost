package sleep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func Wakeup(c *clientgo.GoClient, ns string, force bool) error {
	// 1. check record
	record, err := cluster_user.
		NewClusterUserService().
		GetFirst(context.TODO(), model.ClusterUserModel{Namespace: ns})
	if err != nil {
		return err
	}

	// 2. check namespace
	namespace, err := c.Clientset().
		CoreV1().
		Namespaces().
		Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 3. check replicas
	if namespace.Annotations == nil || len(namespace.Annotations[KReplicas]) == 0 {
		return errors.New(fmt.Sprintf("cannot find `%s` from annotations", KReplicas))
	}
	var replicas map[string]int32
	err = json.Unmarshal([]byte(namespace.Annotations[KReplicas]), &replicas)
	if err != nil {
		return err
	}

	// 4. restore Deployment
	deps, err := c.Clientset().
		AppsV1().
		Deployments(ns).
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
				Deployments(ns).
				Update(context.TODO(), &dp, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 5. restore StatefulSet
	sets, err := c.Clientset().
		AppsV1().
		StatefulSets(ns).
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
				StatefulSets(ns).
				Update(context.TODO(), &st, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 6. restore CronJob
	jobs, err := c.Clientset().
		BatchV1beta1().
		CronJobs(ns).
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
			CronJobs(ns).
			Update(context.TODO(), &jb, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// 7. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KStatus:      KActive,
				KForceSleep:  "",
				KForceWakeup: ternary(force, timestamp(), "").(string),
			},
		},
	})
	_, err = c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// 8. write to database
	return cluster_user.
		NewClusterUserService().
		Modify(context.TODO(), record.ID, map[string]interface{}{
			"sleep_at":  nil,
			"is_asleep": false,
		})
}
