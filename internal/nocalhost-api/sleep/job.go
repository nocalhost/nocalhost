package sleep

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"time"
)

var Job = &model.Job{
	Spec: func() string {
		return fmt.Sprintf("@every %s", viper.GetString("sleep.cron"))
	},
	Exec: func() {
		// 1. obtain clusters
		clusters, err := cluster.NewClusterService().GetList(context.TODO())
		if err != nil {
			log.Errorf("Failed to resolve cluster list, err: %v", err)
			return
		}
		for _, it := range clusters {
			// 2. skip this cluster if `InspectAt` has not expired
			if it.InspectAt == nil || isExpired(*it.InspectAt) {
				go execCluster(it)
			}
		}
	},
}

func execCluster(cs *model.ClusterList) {
	// 1. recover
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered, cluster: %s, err: %v", cs.ClusterName, err)
		}
	}()
	// 2. create lock
	now := time.Now().UTC()
	_, err := cluster.NewClusterService().Update(context.TODO(), map[string]interface{}{
		"InspectAt": &now,
	}, cs.ID)
	if err != nil {
		log.Errorf("Failed to create lock, cluster: %s, err: %v", cs.ClusterName, err)
		return
	}
	// 3. init client-go
	client, err := clientgo.NewAdminGoClient([]byte(cs.KubeConfig))
	if err != nil {
		log.Errorf("Failed to resolve client-go, err: %v", err)
		return
	}
	// 4. obtain namespaces
	namespaces, err := client.Clientset().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to resolve namespace list, cluster: %s, err: %v", cs.ClusterName, err)
		return
	}
	// 5. check namespace
	for _, ns := range namespaces.Items {
		execNamespace(client, &ns)
	}
	// 6. remove lock
	_, err = cluster.NewClusterService().Update(context.TODO(), map[string]interface{}{
		"InspectAt": nil,
	}, cs.ID)
	if err != nil {
		log.Errorf("Failed to remove lock, cluster: %s, err: %v", cs.ClusterName, err)
		return
	}
}

func execNamespace(c *clientgo.GoClient, ns *v1.Namespace) {
	// 1. inspect
	act, err := Inspect(ns)
	if err != nil {
		log.Errorf("Failed to call `ShouldSleep`, ns: %s, err: %v", ns.Name, err)
		return
	}
	// 2. ToBeAsleep
	if act == ToBeAsleep {
		err = Asleep(c, ns.Name, false)
		if err != nil {
			log.Errorf("Failed to sleep, ns: %s, err: %v", ns.Name, err)
			return
		}
		log.Infof("Sleep, ns: %s", ns.Name)
	}
	// 3. ToBeWakeup
	if act == ToBeWakeup {
		err = Wakeup(c, ns.Name, false)
		if err != nil {
			log.Errorf("Failed to wakeup, ns: %s, err: %v", ns.Name, err)
			return
		}
		log.Infof("Wakeup, ns: %s", ns.Name)
	}
}

func isExpired(other time.Time) bool {
	return time.Now().Sub(other) > 5*time.Minute
}
