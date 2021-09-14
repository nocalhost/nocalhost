/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	//"github.com/syncthing/syncthing/lib/discover"
	"nocalhost/pkg/nhctl/clientgoutils"
	"time"
)

var timeout int64

func init() {
	checkClusterCmd.Flags().Int64Var(&timeout, "timeout", 5, "timeout duration of checking cluster")
	checkCmd.AddCommand(checkClusterCmd)
}

var checkClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "check k8s cluster's status",
	Long:  `check k8s cluster's status`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonResp := &struct {
			Code int    `json:"code"`
			Info string `json:"info"`
		}{}
		defer func() {
			bys, _ := json.Marshal(jsonResp)
			fmt.Println(string(bys))
		}()

		err := checkClusterAvailable(kubeConfig, time.Duration(timeout)*time.Second)
		if err != nil {
			jsonResp.Code = 201
			jsonResp.Info = err.Error()
			return
		}
		jsonResp.Code = 200
		jsonResp.Info = "Connected successfully"
	},
}

func checkClusterAvailable(kube string, timeout time.Duration) error {
	//client, err := daemon_client.NewDaemonClient(false)
	//must(err)
	//
	//daemonServerInfo, err := client.SendGetDaemonServerInfoCommand()
	//must(err)
	//
	//marshal, err := json.Marshal(daemonServerInfo)
	//must(err)
	var errChan = make(chan error, 1)
	go func() {
		c, err := clientgoutils.NewClientGoUtils(kube, "")
		if err != nil {
			errChan <- err
			return
		}
		_, err = c.ClientSet.ServerVersion()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
		return errors.New(fmt.Sprintf("Check cluster available timeout after %s", timeout.String()))
	}
}
