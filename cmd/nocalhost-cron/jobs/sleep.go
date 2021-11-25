package jobs

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"time"

	"nocalhost/internal/nocalhost-api/sleep"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

var Sleep = &Job{
	Spec: "@every 30s",
	Task: task,
}

func task() {
	// 1. obtain clusters
	clusters, err := cluster.NewClusterService().GetList(context.TODO())
	if err != nil {
		log.Errorf("Failed to resolve cluster list, err: %v", err)
		return
	}
	for _, cs := range clusters {
		go func() {
			// 2. init client-go
			client, err := clientgo.NewAdminGoClient([]byte(cs.KubeConfig))
			if err != nil {
				log.Errorf("Failed to resolve client-go, err: %v", err)
				return
			}
			// 3. obtain namespaces
			namespaces, err := client.Clientset().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				log.Errorf("Failed to resolve namespace list, cluster: %s, err: %v", cs.ClusterName, err)
				return
			}
			for _, ns := range namespaces.Items {
				// 4. exec
				exec(client, &ns)
			}
		}()
	}
}

func exec(c *clientgo.GoClient, ns *v1.Namespace) {
	// 1. inspect
	act, err := sleep.Inspect(ns)
	if err != nil {
		log.Errorf("Failed to call `ShouldSleep`, ns: %s, err: %v", ns.Name, err)
		return
	}
	// 2. should sleep
	if act == sleep.ToBeAsleep {
		// 4. check database
		record, err := cluster_user.NewClusterUserService().GetFirst(context.TODO(), model.ClusterUserModel{Namespace: ns.Name})
		if err != nil {
			log.Errorf("Failed to resolve record, ns: %s, err: %v", ns.Name, err)
			return
		}
		// 3. exec sleep
		err = sleep.Sleep(c, ns.Name, false)
		if err != nil {
			log.Errorf("Failed to sleep, ns: %s, err: %v", ns.Name, err)
			return
		}
		// 4. write to database
		now := time.Now().UTC()
		err = cluster_user.
			NewClusterUserService().
			Modify(context.TODO(), record.ID, map[string]interface{}{
				"SleepAt":  &now,
				"IsAsleep": true,
			})
		if err != nil {
			log.Errorf("Failed to update database, ns: %s, err: %v", ns.Name, err)
			return
		}
		log.Infof("Sleep, ns: %s", ns.Name)
	}
	// 5. should wakeup
	if act == sleep.ToBeWakeup {
		// 6. check database
		record, err := cluster_user.NewClusterUserService().GetFirst(context.TODO(), model.ClusterUserModel{Namespace: ns.Name})
		if err != nil {
			log.Errorf("Failed to resolve record, ns: %s, err: %v", ns.Name, err)
			return
		}
		// 7. exec wakeup
		err = sleep.Wakeup(c, ns.Name, false)
		if err != nil {
			log.Errorf("Failed to wakeup, ns: %s, err: %v", ns.Name, err)
			return
		}
		// 8. update database
		err = cluster_user.
			NewClusterUserService().
			Modify(context.TODO(), record.ID, map[string]interface{}{
				"SleepAt":  nil,
				"IsAsleep": false,
			})
		if err != nil {
			log.Errorf("Failed to update database, ns: %s, err: %v", ns.Name, err)
			return
		}
		log.Infof("Wakeup, ns: %s", ns.Name)
	}
}
