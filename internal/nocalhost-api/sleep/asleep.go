package sleep

import (
	"context"
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"time"
)

func Asleep(c *clientgo.GoClient, ns string, force bool) error {
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
	var replicas map[string]int32
	if namespace.Annotations == nil || len(namespace.Annotations[KReplicas]) == 0 {
		replicas = make(map[string]int32)
	} else {
		err = json.Unmarshal([]byte(namespace.Annotations[KReplicas]), &replicas)
		if err != nil {
			return err
		}
	}

	// 4. suspend CronJob
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
		jb.Spec.Suspend = &exactly
		_, err = c.Clientset().
			BatchV1beta1().
			CronJobs(ns).
			Update(context.TODO(), &jb, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// 5. purging Deployment
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
		if count := evaluate(dp.Spec.Replicas); count > 0 {
			replicas[KDeployment+":"+dp.Name] = count
			dp.Spec.Replicas = &zero
			_, err = c.Clientset().
				AppsV1().
				Deployments(ns).
				Update(context.TODO(), &dp, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 6. purging StatefulSet
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
		if count := evaluate(st.Spec.Replicas); count > 0 {
			replicas[KStatefulSet+":"+st.Name] = count
			st.Spec.Replicas = &zero
			_, err = c.Clientset().
				AppsV1().
				StatefulSets(ns).
				Update(context.TODO(), &st, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 7. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KStatus:      KAsleep,
				KReplicas:    stringify(replicas),
				KForceSleep:  ternary(force, timestamp(), "").(string),
				KForceWakeup: "",
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
	now := time.Now().UTC()
	return cluster_user.
		NewClusterUserService().
		Modify(context.TODO(), record.ID, map[string]interface{}{
			"sleep_at":  &now,
			"is_asleep": true,
		})
}

func evaluate(val *int32) int32 {
	if val == nil {
		return 1
	}
	return *val
}
