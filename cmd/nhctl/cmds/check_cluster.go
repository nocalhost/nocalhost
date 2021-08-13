/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/clientgoutils"
)

func init() {
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

		err := checkClusterAvailable(kubeConfig)
		if err != nil {
			jsonResp.Code = 201
			jsonResp.Info = err.Error()
			return
		}
		//_, err = c.ClientSet.ServerVersion()
		//if err != nil {
		//	jsonResp.Code = 201
		//	jsonResp.Info = err.Error()
		//	return
		//}
		jsonResp.Code = 200
		jsonResp.Info = "Connected successfully"
	},
}

func checkClusterAvailable(kube string) error {
	c, err := clientgoutils.NewClientGoUtils(kube, "")
	if err != nil {
		return err
	}
	if _, err = c.ClientSet.ServerVersion(); err != nil {
		return err
	}
	return nil
}
