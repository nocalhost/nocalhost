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
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset application",
	Long:  `reset application`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if err := Prepare(); err != nil {
			log.FatalE(err, "")
		}

		if len(args) > 0 {
			applicationName := args[0]
			if applicationName != "" {
				if applicationName == nocalhost.DefaultNocalhostApplication {
					log.Error(nocalhost.DefaultNocalhostApplicationOperateErr)
					return
				}
				resetApplication(applicationName)
				return
			}
		}

		// Reset all applications under specify namespace
		metas, err := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
		mustI(err, "Failed to get applications")
		for _, meta := range metas {
			resetApplication(meta.Application)
		}
	},
}

func resetApplication(applicationName string) {
	var err error
	initApp(applicationName)
	// Stop BackGroup Process
	appProfile, _ := nocalhostApp.GetProfile()
	for _, profile := range appProfile.SvcProfile {
		nhSvc := initService(profile.ActualName, profile.Type)
		if nhSvc.IsInDevMode() {
			utils.Should(nhSvc.StopSyncAndPortForwardProcess(true))
		} else if len(profile.DevPortForwardList) > 0 {
			utils.Should(nhSvc.StopAllPortForward())
		}
	}

	// Remove files
	time.Sleep(1 * time.Second)
	if err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace); err != nil {
		log.WarnE(err, "")
	} else {
		log.Info("Files have been clean up")
	}
	log.Infof("Application %s has been reset.\n", applicationName)
}
