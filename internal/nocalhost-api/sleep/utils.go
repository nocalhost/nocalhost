package sleep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"strconv"
	"time"
)

const (
	kActive = "active"
	kAsleep = "asleep"
	kDeployment = "Deployment"
	kStatefulSet = "StatefulSet"
	kConfig = "nocalhost.dev.sleep/config"
	kStatus = "nocalhost.dev.sleep/status"
	kReplicas = "nocalhost.dev.sleep/replicas"
	kForceSleep = "nocalhost.dev.sleep/force-sleep"
	kForceWakeup = "nocalhost.dev.sleep/force-wakeup"
)

func Sleep(c* clientgo.GoClient, ns string, force bool) error {
	// 1. check namespace
	namespace, err := c.Clientset().
		CoreV1().
		Namespaces().
		Get(context.TODO(), ns, metav1.GetOptions{})

	if err != nil {
		return err
	}

	var replicas map[string]int32
	if namespace.Annotations == nil || len(namespace.Annotations[kReplicas]) == 0 {
		replicas = make(map[string]int32)
	} else {
		err = json.Unmarshal([]byte(namespace.Annotations[kReplicas]), &replicas)
		if err != nil {
			return err
		}
	}

	// 2. purging Deployment
	var zero int32 = 0
	dps, err := c.Clientset().
		AppsV1().
		Deployments(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dp := range dps.Items {
		if *dp.Spec.Replicas > 0 {
			replicas[kDeployment + ":" + dp.Name] = *dp.Spec.Replicas
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
	// 3. purging StatefulSet
	sts, err := c.Clientset().
		AppsV1().
		StatefulSets(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, st := range sts.Items {
		if *st.Spec.Replicas > 0 {
			replicas[kStatefulSet + ":" + st.Name] = *st.Spec.Replicas
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
	// 4. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				kStatus: kAsleep,
				kReplicas: stringify(replicas),
				kForceSleep: ternary(force, timestamp(), "").(string),
				kForceWakeup: "",
			},
		},
	})
	_, err = c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

func Wakeup(c* clientgo.GoClient, ns string, force bool) error {
	// 1. check namespace
	namespace, err := c.Clientset().
		CoreV1().
		Namespaces().
		Get(context.TODO(), ns, metav1.GetOptions{})

	if err != nil {
		return err
	}

	if namespace.Annotations == nil || len(namespace.Annotations[kReplicas]) == 0 {
		return errors.New(fmt.Sprintf("cannot find `%s` from annotations", kReplicas))
	}

	var replicas map[string]int32
	err = json.Unmarshal([]byte(namespace.Annotations[kReplicas]), &replicas)
	if err != nil {
		return err
	}

	// 2. restore Deployment
	dps, err := c.Clientset().
		AppsV1().
		Deployments(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dp := range dps.Items {
		n, ok := replicas[kDeployment + ":" + dp.Name]
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
	// 3. restore StatefulSet
	sts, err := c.Clientset().
		AppsV1().
		StatefulSets(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, st := range sts.Items {
		n, ok := replicas[kDeployment + ":" + st.Name]
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
	// 4. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				kStatus: kActive,
				kForceSleep: "",
				kForceWakeup: ternary(force, timestamp(), "").(string),
			},
		},
	})
	_, err = c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

func CreateSleepConfig(c *clientgo.GoClient, ns string, config string) error {
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				kConfig: config,
				kStatus: "",
			},
		},
	})

	_, err := c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

func DeleteSleepConfig(c *clientgo.GoClient, ns string) error {
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				kConfig: "",
				kStatus: "",
			},
		},
	})

	_, err := c.Clientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), ns, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

func stringify(v interface{}) string {
	marshal, _ := json.Marshal(v)
	return string(marshal)
}

func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func ternary(a bool, b, c interface{}) interface{} {
	if a {
		return b
	}
	return c
}
