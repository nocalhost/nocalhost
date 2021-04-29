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
	"io/ioutil"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"sigs.k8s.io/yaml"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var outputType string

const JSON = "json"

func init() {
	getCmd.PersistentFlags().StringVarP(
		&appName, "application", "a", "", "application name",
	)
	getCmd.PersistentFlags().StringVarP(
		&outputType, "outputType", "o", JSON, "json or yaml",
	)
	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get type",
	Short: "Get resource info",
	Long: `Get resource info
Examples: 
  # Get all application
  nhctl get application --kubeconfig=kubeconfigfile

  # Get all application in namespace
  nhctl get application -n namespaceName --kubeconfig=kubeoconfigpath
  
  # Get all deployment of application in namespace
  nhctl get deployment -n namespaceName -a bookinfo --kubeconfig=kubeconfigpath

usage: nhctl get service serviceName [-n namespace] --kubeconfig=kubeconfigfile

`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		resourceName := ""
		if len(args) >= 2 {
			resourceName = args[1]
		}
		if appName != "" {
			initApp(appName)
		}
		bytes, err := ioutil.ReadFile(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}
		cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
		if err != nil {
			log.Fatal(err)
		}
		appMeta, err := cli.SendGetResourceInfoCommand(string(bytes), nameSpace, appName, resourceType, resourceName)
		if appMeta != nil && err == nil {
			var b []byte
			var err error
			if JSON == outputType {
				b, err = json.Marshal(appMeta)
			} else {
				b, err = yaml.Marshal(appMeta)
			}
			if err == nil {
				fmt.Print(string(b))
			}
		}
	},
}
