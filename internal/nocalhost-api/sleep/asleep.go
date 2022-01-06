package sleep

import (
	"context"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func Asleep(c *clientgo.GoClient, s *model.ClusterUserModel, force bool) error {
	// 1. sleep
	if err := _asleep(c.Config, c.Clientset(), s.Namespace, force); err != nil {
		return err
	}

	// 2. write to database
	now := time.Now().UTC()
	return cluster_user.
		NewClusterUserService().
		Modify(context.TODO(), s.ID, map[string]interface{}{
			"sleep_at":     &now,
			"sleep_status": KAsleep,
		})
}

func _asleep(config []byte, c kubernetes.Interface, namespace string, force bool) error {
	// 1. check ns
	ns, err := c.CoreV1().
		Namespaces().
		Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 2. check replicas
	var replicas map[string]int32
	if ns.Annotations == nil || len(ns.Annotations[KReplicas]) == 0 {
		replicas = make(map[string]int32)
	} else {
		err = json.Unmarshal([]byte(ns.Annotations[KReplicas]), &replicas)
		if err != nil {
			return err
		}
	}

	// 3. suspend CronJob
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
		jb.Spec.Suspend = &exactly
		_, err = c.BatchV1beta1().
			CronJobs(namespace).
			Update(context.TODO(), &jb, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// 4. purging Deployment
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
		if count := evaluate(dp.Spec.Replicas); count > 0 {
			replicas[KDeployment+":"+dp.Name] = count
			dp.Spec.Replicas = &zero
			_, err = c.AppsV1().
				Deployments(namespace).
				Update(context.TODO(), &dp, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 5. purging StatefulSet
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
		if count := evaluate(st.Spec.Replicas); count > 0 {
			replicas[KStatefulSet+":"+st.Name] = count

			// sleep vcluster
			if st.GetLabels()["app"] == "vcluster" {
				if err := sleepVCluster(namespace, config, c, force); err != nil {
					return err
				}
				continue
			}

			st.Spec.Replicas = &zero
			_, err = c.AppsV1().
				StatefulSets(namespace).
				Update(context.TODO(), &st, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// 6. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				KReplicas:    stringify(replicas),
				KSleepStatus: KAsleep,
				KForceAsleep: ternary(force, timestamp(), "").(string),
				KForceWakeup: "",
			},
		},
	})
	_, err = c.CoreV1().
		Namespaces().
		Patch(context.TODO(), namespace, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func evaluate(val *int32) int32 {
	if val == nil {
		return 1
	}
	return *val
}
