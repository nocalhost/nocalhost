package sleep

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
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
	err := cluster.NewClusterService().Lockup(context.TODO(), cs.ID, cs.InspectAt)
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
	// 4. obtain dev spaces
	spaces, err := cluster_user.NewClusterUserService().GetList(context.TODO(), model.ClusterUserModel{})
	if err != nil {
		log.Errorf("Failed to resolve namespace list, cluster: %s, err: %v", cs.ClusterName, err)
		return
	}
	// 5. check dev spaces
	for _, s := range spaces {
		if s.Namespace != "*" {
			execDevSpace(client, s)
		}
	}
	// 6. remove lock
	err = cluster.NewClusterService().Unlock(context.TODO(), cs.ID)
	if err != nil {
		log.Errorf("Failed to remove lock, cluster: %s, err: %v", cs.ClusterName, err)
		return
	}
}

func execDevSpace(c *clientgo.GoClient, s *model.ClusterUserModel) {
	// 1. inspect
	act, err := Inspect(c, s)
	if err != nil {
		log.Errorf("Failed to call `Inspect`, ns: %s, err: %v", s.Namespace, err)
		return
	}
	// 2. ToBeAsleep
	if act == ToBeAsleep {
		err = Asleep(c, s, false)
		if err != nil {
			log.Errorf("Failed to call `Asleep`, ns: %s, err: %v", s.Namespace, err)
			return
		}
		log.Infof("Sleep, ns: %s", s.Namespace)
	}
	// 3. ToBeWakeup
	if act == ToBeWakeup {
		err = Wakeup(c, s, false)
		if err != nil {
			log.Errorf("Failed to wakeup, ns: %s, err: %v", s.Namespace, err)
			return
		}
		log.Infof("Wakeup, ns: %s", s.Namespace)
	}
}

func isExpired(other time.Time) bool {
	return time.Now().Sub(other) > 5*time.Minute
}
