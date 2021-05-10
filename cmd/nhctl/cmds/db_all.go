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
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"
)

func init() {
	dbAllCmd.Flags().StringVar(&appName, "app", "", "List leveldb data of specified application")
	//pvcListCmd.Flags().StringVar(&pvcFlags.Svc, "controller", "", "List PVCs of specified service")
	dbCmd.AddCommand(dbAllCmd)
}

var dbAllCmd = &cobra.Command{
	Use:   "all [NAME]",
	Short: "Get all leveldb data",
	Long:  `Get all leveldb data`,
	Run: func(cmd *cobra.Command, args []string) {
		result, err := nocalhost.ListAllFromApplicationDb(nameSpace, appName)
		must(err)
		for key, val := range result {
			fmt.Printf("%s=%s\n", key, val)
		}
	},
}
