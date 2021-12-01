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
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
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
	// 1. check annotations
	if ns.Annotations == nil {
		return ToBeIgnore, nil
	}
	// 2. check force sleep
	if len(ns.Annotations[kForceSleep]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[kForceSleep]), 0).UTC()
		if t.Month() == now.Month() && t.Day() == now.Day() {
			return ToBeIgnore, nil
		}
	}
	// 3. check force wakeup
	if len(ns.Annotations[kForceWakeup]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[kForceWakeup]), 0).UTC()
		if t.Month() == now.Month() && t.Day() == now.Day() {
			return ToBeIgnore, nil
		}
	}
	// 4. check sleep config
	if len(ns.Annotations[kConfig]) == 0 {
		if ns.Annotations[kStatus] == kAsleep {
			return ToBeWakeup, nil
		}
		return ToBeIgnore, nil
	}
	// 5. parse sleep config
	var conf model.SleepConfig
	err := json.Unmarshal([]byte(ns.Annotations[kConfig]), &conf)
	if err != nil {
		return ToBeIgnore, err
	}
	if len(conf.ByWeek) == 0 {
		return ToBeWakeup, nil
	}
	// 6. match sleep config
	for _, f := range conf.ByWeek {
		now := time.Now().In(f.TimeZone())
		d1 := time.Duration(*f.SleepDay - now.Weekday())
		d2 := time.Duration(*f.WakeupDay - now.Weekday())

		if *f.WakeupDay < *f.SleepDay {
			d2 = time.Duration(time.Saturday - *f.SleepDay + *f.WakeupDay + 1)
		}
		// sleep time
		t1 := now.Add(d1 * 24 * time.Hour)
		t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), f.Hour(f.SleepTime), f.Minute(f.SleepTime), 0, 0, f.TimeZone())
		// wakeup time
		t2 := now.Add(d2 * 24 * time.Hour)
		t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), f.Hour(f.WakeupTime), f.Minute(f.WakeupTime), 0, 0, f.TimeZone())

		println(ns.Name, " Sleep:【" + t1.String() + "】", "Wakeup:【" + t2.String() + "】")

		if now.After(t1) && now.Before(t2) {
			if ns.Annotations[kStatus] == kAsleep {
				return ToBeIgnore, nil
			}
			return ToBeAsleep, nil
		}
	}
	// 7. there are no matching rules, then dev space need to be woken up.
	if ns.Annotations[kStatus] == kActive {
		return ToBeIgnore, nil
	}
	return ToBeWakeup, nil
}

func Calc(items *[]model.ByWeek) float32 {
	var x [10080]uint8
	for _, it := range *items {
		a := it.ToInt(*it.SleepDay, it.SleepTime)
		b := it.ToInt(*it.WakeupDay, it.WakeupTime)
		// extend into next week
		if b < a {
			for i := a; i < 10080; i++ {
				x[i] =1
			}
			for i := 0; i < b; i++ {
				x[i] = 1
			}
		} else {
			for i := a; i < b; i++ {
				x[i] = 1
			}
		}
	}

	var c float32 = 0
	for _, v := range x {
		if v == 1 {
			c++
		}
	}
	return c / 10080
}

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
	if namespace.Annotations == nil || len(namespace.Annotations[kReplicas]) == 0 {
		replicas = make(map[string]int32)
	} else {
		err = json.Unmarshal([]byte(namespace.Annotations[kReplicas]), &replicas)
		if err != nil {
			return err
		}
	}
	// 4. purging CronJob
	jobs, err := c.Clientset().
		BatchV1beta1().
		CronJobs(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, jb := range jobs.Items {
		suspend := true
		jb.Spec.Suspend = &suspend
		_, err = c.Clientset().
			BatchV1beta1().
			CronJobs(ns).
			Update(context.TODO(), &jb, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	// 5. purging Deployment
	dps, err := c.Clientset().
		AppsV1().
		Deployments(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dp := range dps.Items {
		var count int32 = 0
		if dp.Spec.Replicas == nil {
			count = 1
		} else {
			count = *dp.Spec.Replicas
		}
		if count > 0 {
			replicas[kDeployment + ":" + dp.Name] = count
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
	sts, err := c.Clientset().
		AppsV1().
		StatefulSets(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, st := range sts.Items {
		var count int32 = 0
		if st.Spec.Replicas == nil {
			count = 1
		} else {
			count = *st.Spec.Replicas
		}
		if count > 0 {
			replicas[kStatefulSet + ":" + st.Name] = count
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

func Wakeup(c* clientgo.GoClient, ns string, force bool) error {
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
	if namespace.Annotations == nil || len(namespace.Annotations[kReplicas]) == 0 {
		return errors.New(fmt.Sprintf("cannot find `%s` from annotations", kReplicas))
	}
	var replicas map[string]int32
	err = json.Unmarshal([]byte(namespace.Annotations[kReplicas]), &replicas)
	if err != nil {
		return err
	}
	// 4. restore Deployment
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
	// 5. restore StatefulSet
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
	// 6. restore CronJob
	jobs, err := c.Clientset().
		BatchV1beta1().
		CronJobs(ns).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, jb := range jobs.Items {
		suspend := false
		jb.Spec.Suspend = &suspend
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

func ApplySleepConfig(c *clientgo.GoClient, id uint64, ns string, conf model.SleepConfig) (*model.ClusterUserModel, error) {
	// 1. update annotations
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				kConfig: ternary(len(conf.ByWeek) == 0, "" , stringify(conf)).(string),
				kStatus: "",
				kForceSleep: "",
				kForceWakeup: "",
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
