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
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/daemon_client"

	//"github.com/syncthing/syncthing/lib/discover"
	"nocalhost/pkg/nhctl/clientgoutils"
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

		err := checkClusterAvailable(common.KubeConfig)
		if err != nil {
			jsonResp.Code = 201
			jsonResp.Info = err.Error()
			return
		}
		jsonResp.Code = 200
		jsonResp.Info = "Connected successfully"
	},
}

func checkClusterAvailable(kube string) error {

	kubeContent, err := clientgoutils.GetKubeContentFromPath(kube)
	if err != nil {
		return err
	}

	client, err := daemon_client.GetDaemonClient(false)
	if err != nil {
		return err
	}

	checkClusterStatus, err := client.SendCheckClusterStatusCommand(string(kubeContent))
	if err != nil {
		return err
	}
	if checkClusterStatus.Available == false {
		return errors.New(checkClusterStatus.Info)
	}
	return nil
}
