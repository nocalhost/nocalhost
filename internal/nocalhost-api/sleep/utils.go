package sleep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cast"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"strconv"
	"time"
)

type ToBe int

const (
	ToBeIgnore ToBe = iota
	ToBeWakeup
	ToBeAsleep
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

var zero int32 = 0

func Inspect(ns *v1.Namespace) (ToBe, error) {
	if ns.Annotations == nil {
		return ToBeIgnore, nil
	}
	if len(ns.Annotations[kConfig]) == 0 {
		return ToBeIgnore, nil
	}
	if len(ns.Annotations[kForceWakeup]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[kForceWakeup]), 0).UTC()
		if t.Month() == now.Month() && t.Day() == now.Day() {
			return ToBeIgnore, nil
		}
	}
	var conf model.SleepConfig
	err := json.Unmarshal([]byte(ns.Annotations[kConfig]), &conf)
	if err != nil {
		return ToBeIgnore, err
	}
	if len(conf.Schedules) == 0 {
		return ToBeWakeup, nil
	}
	for _, f := range conf.Schedules {
		now := time.Now().In(f.TimeZone())
		d1 := time.Duration(*f.SleepDay - now.Weekday())
		d2 := time.Duration(*f.WakeupDay - now.Weekday())

		if *f.WakeupDay < *f.SleepDay {
			d2 = time.Duration(time.Saturday - *f.SleepDay + *f.WakeupDay + 1)
		}

		t1 := now.Add(d1 * 24 * time.Hour)
		t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), f.Hour(f.SleepTime), f.Minute(f.SleepTime), 0, 0, f.TimeZone())

		t2 := now.Add(d2 * 24 * time.Hour)
		t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), f.Hour(f.WakeupTime), f.Minute(f.WakeupTime), 0, 0, f.TimeZone())

		if now.After(t1) && now.Before(t2) {
			if ns.Annotations[kStatus] == kAsleep {
				return ToBeIgnore, nil
			}
			println(ns.Name, " Sleep:【" + t1.String() + "】", "Wakeup:【" + t2.String() + "】")
			return ToBeAsleep, nil
		}
	}
	if ns.Annotations[kStatus] == kActive {
		return ToBeIgnore, nil
	}
	return ToBeWakeup, nil
}

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
		n, ok := replicas[kStatefulSet + ":" + st.Name]
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
