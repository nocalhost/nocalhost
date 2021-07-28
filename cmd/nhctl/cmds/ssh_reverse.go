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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

var (
	podName    string
	podSshPort int
	remotePort int
	localPort  int
)

func init() {
	sshRCmd.PersistentFlags().StringVarP(&podName, "pod", "", "", "pod name witch has ssh server")
	sshRCmd.PersistentFlags().IntVarP(&podSshPort, "sshport", "p", 50022, "pod ssh service port")
	sshRCmd.PersistentFlags().IntVarP(&remotePort, "remote", "", 2222, "remote port")
	sshRCmd.PersistentFlags().IntVarP(&localPort, "local", "", 2222, "local port")
	sshCmd.AddCommand(sshRCmd)
}

var sshRCmd = &cobra.Command{
	Use:     "reverse",
	Short:   "ssh reverse",
	Long:    `ssh reverse`,
	Example: `ssh reverse`,
	Run: func(cmd *cobra.Command, args []string) {

		port, _ := ports.GetAvailablePort()
		app := &app.Application{
			NameSpace:  nameSpace,
			KubeConfig: kubeConfig,
		}
		okchan := make(chan struct{})
		go func() {
			if err := app.PortForwardFollow(podName, port, podSshPort, okchan); err != nil {
				log.Fatal(err)
			}
		}()
		<-okchan
		err := utils.Reverse(
			utils.DefaultRoot,
			fmt.Sprintf("127.0.0.1:%d", port),
			fmt.Sprintf("0.0.0.0:%d", remotePort),
			fmt.Sprintf("127.0.0.1:%d", localPort),
		)
		if err != nil {
			log.Fatalf("error while create reverse tunnel, err: %v", err)
		}
	},
}
