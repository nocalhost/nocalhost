/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
		c, err := clientgoutils.NewClientGoUtils(kubeConfig, "")
		if err != nil {
			jsonResp.Code = 201
			jsonResp.Info = err.Error()
			return
		}
		_, err = c.ClientSet.ServerVersion()
		if err != nil {
			jsonResp.Code = 201
			jsonResp.Info = err.Error()
			return
		}
		jsonResp.Code = 200
		jsonResp.Info = "Connected successfully"
	},
}
