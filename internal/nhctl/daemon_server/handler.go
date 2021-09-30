/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/clientgoutils"
	"sync"
	"time"
)

//type ClusterStatus struct {
//	Available bool
//	Info      string
//}

// key: string
// val: *daemon_common.CheckClusterStatus
var clusterStatusMap sync.Map

func HandleCheckClusterStatus(cmd *command.CheckClusterStatusCommand) (*daemon_common.CheckClusterStatus, error) {
	kubeDir := nocalhost.GetOrGenKubeConfigPath(cmd.KubeConfigContent)

	c, ok := clusterStatusMap.Load(kubeDir)
	if ok {
		s := c.(*daemon_common.CheckClusterStatus)
		return s, nil
	}

	err := checkClusterStatus(kubeDir, 5*time.Second)
	var ccs *daemon_common.CheckClusterStatus
	if err != nil {
		ccs = &daemon_common.CheckClusterStatus{Available: false, Info: err.Error()}
	} else {
		ccs = &daemon_common.CheckClusterStatus{Available: true}
	}
	clusterStatusMap.Store(kubeDir, ccs)
	return ccs, nil
}

func checkClusterStatus(kubePath string, timeout time.Duration) error {
	var errChan = make(chan error, 1)
	go func() {
		c, err := clientgoutils.NewClientGoUtils(kubePath, "")
		if err != nil {
			errChan <- err
			return
		}
		_, err = c.ClientSet.ServerVersion()
		errChan <- err
	}()

	var err error
	select {
	case err = <-errChan:
	case <-time.After(timeout):
		err = errors.New("Check cluster available timeout after 5s")
	}
	return err
}

func checkClusterStatusCronJob() {
	// todo: add a recover
	for {
		var wg sync.WaitGroup
		clusterStatusMap.Range(func(k, v interface{}) bool {
			wg.Add(1)
			go func(key, value interface{}) {
				defer wg.Done()
				css := value.(*daemon_common.CheckClusterStatus)
				if err := checkClusterStatus(key.(string), time.Minute); err != nil {
					css.Available = false
					css.Info = err.Error()
				} else {
					css.Available = true
					css.Info = ""
				}
			}(k, v)
			return true
		})
		wg.Wait()
		<-time.After(time.Minute)
	}

}
