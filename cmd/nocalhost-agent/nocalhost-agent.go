package main

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"time"

	"nocalhost/internal/nocalhost-api/sleep"
	"nocalhost/pkg/nocalhost-agent/tools"
	"nocalhost/pkg/nocalhost-api/conf"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

var (
	config = pflag.StringP("config", "c", "", "config file path.")
)

func main() {
	pflag.Parse()
	// init config
	if err := conf.Init(*config); err != nil {
		panic(err)
	}
	// init logger
	conf.InitLog()
	// init database
	model.Init()
	// migrate database
	model.MigrateDB()
	// init cron
	c := cron.New()
	_, err := c.AddFunc("@every 30s", func() {
		go handler()
	})
	if err != nil {
		log.Errorf("Failed to start nocalhost-agent, err: %v", err)
		return
	}
	c.Start()

	g := tools.Graceful{}
	g.Wait()
}

func handler() {
	// 1. obtain clusters
	clusters, err := cluster.NewClusterService().GetList(context.TODO())
	if err != nil {
		log.Errorf("Failed to resolve cluster list, err: %v", err)
		return
	}
	for _, cs := range clusters {
		client, err := clientgo.NewAdminGoClient([]byte(cs.KubeConfig))
		if err != nil {
			log.Errorf("Failed to resolve client-go, err: %v", err)
			continue
		}
		// 2. obtain namespaces
		namespaces, err := client.Clientset().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("Failed to resolve namespace list, cluster: %s, err: %v", cs.ClusterName, err)
			continue
		}
		for _, ns := range namespaces.Items {
			act, err := sleep.Inspect(&ns)
			if err != nil {
				log.Errorf("Failed to call `ShouldSleep`, ns: %s, err: %v", ns.Name, err)
				continue
			}
			// 3. should sleep
			if act == sleep.ToBeSleep {
				// 4. check database
				record, err := cluster_user.NewClusterUserService().GetFirst(context.TODO(), model.ClusterUserModel{Namespace: ns.Name})
				if err != nil {
					log.Errorf("Failed to resolve record, ns: %s, err: %v", ns.Name, err)
					continue
				}
				// 5. exec sleep
				err = sleep.Sleep(client, ns.Name, false)
				if err != nil {
					log.Errorf("Failed to sleep, ns: %s, err: %v", ns.Name, err)
					continue
				}
				// 6. update database
				now := time.Now().UTC()
				err = cluster_user.
					NewClusterUserService().
					Modify(context.TODO(), record.ID, map[string]interface{}{
						"SleepAt": &now,
						"IsAsleep": true,
					})
				if err != nil {
					log.Errorf("Failed to update database, ns: %s, err: %v", ns.Name, err)
					continue
				}
				log.Infof("Sleep, ns: %s", ns.Name)
				continue
			}
			// 7. should wakeup
			if act == sleep.ToBeWakeup {
				// 8. check database
				record, err := cluster_user.NewClusterUserService().GetFirst(context.TODO(), model.ClusterUserModel{Namespace: ns.Name})
				if err != nil {
					log.Errorf("Failed to resolve record, ns: %s, err: %v", ns.Name, err)
					continue
				}
				// 9. exec wakeup
				err = sleep.Wakeup(client, ns.Name, false)
				if err != nil {
					log.Errorf("Failed to wakeup, ns: %s, err: %v", ns.Name, err)
					continue
				}
				// 10. update database
				err = cluster_user.
					NewClusterUserService().
					Modify(context.TODO(), record.ID, map[string]interface{}{
						"SleepAt": nil,
						"IsAsleep": false,
					})
				if err != nil {
					log.Errorf("Failed to update database, ns: %s, err: %v", ns.Name, err)
					continue
				}
				log.Infof("Wakeup, ns: %s", ns.Name)
			}
		}
	}
}

