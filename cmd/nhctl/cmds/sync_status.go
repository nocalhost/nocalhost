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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/syncthing/network/req"
)

var syncStatusOps = &app.SyncStatusOptions{}

func init() {
	//syncStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	syncStatusCmd.Flags().StringVarP(&deployment, "deployment", "d", string(appmeta.Deployment),
		"k8s deployment which your developing service exists",
	)
	syncStatusCmd.Flags().StringVarP(&serviceType, "svc-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.Override, "override", false,
		"override the remote changing according to the local sync folder",
	)

	rootCmd.AddCommand(syncStatusCmd)
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status [NAME]",
	Short: "Files sync status",
	Long:  "Tracing the files sync status, include local folder and remote device",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initApp(applicationName)
		nhSvc := initService(deployment, serviceType)

		if !nhSvc.IsInDevMode() {
			display(req.NotInDevModeTemplate)
			return
		}

		client := nhSvc.NewSyncthingHttpClient()

		if syncStatusOps.Override {
			must(client.FolderOverride())
			display("Succeed")
			return
		}

		display(client.GetSyncthingStatus())
	},
}

func display(v interface{}) {
	marshal, _ := json.Marshal(v)
	fmt.Printf("%s", string(marshal))
}
