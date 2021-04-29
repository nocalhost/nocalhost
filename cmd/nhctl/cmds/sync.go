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
	"context"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	fileSyncCmd.Flags().StringVarP(&serviceType, "svc-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.SyncDouble, "double", "b", false,
		"if use double side sync")
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Resume, "resume", false,
		"resume file sync")
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Stop, "stop", false, "stop file sync")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.SyncedPattern, "synced-pattern", "s", []string{},
		"local synced pattern")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.IgnoredPattern, "ignored-pattern", "i", []string{},
		"local ignored pattern")
	fileSyncCmd.Flags().StringVar(&fileSyncOps.Container, "container", "", "container name of pod to sync")
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Override, "overwrite", true,
		"override the remote changing according to the local sync folder while start up")
	rootCmd.AddCommand(fileSyncCmd)
}

var fileSyncCmd = &cobra.Command{
	Use:   "sync [NAME]",
	Short: "Sync files to remote Pod in Kubernetes",
	Long:  `Sync files to remote Pod in Kubernetes`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]

		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if !nocalhostSvc.IsInDevMode() {
			log.Fatalf("Service \"%s\" is not in developing", deployment)
		}

		// resume port-forward and syncthing
		if fileSyncOps.Resume || fileSyncOps.Stop {
			utils.ShouldI(nocalhostSvc.StopFileSyncOnly(), "Error occurs when stopping sync process")
			if fileSyncOps.Stop {
				return
			}
		}

		podName, err := nocalhostSvc.GetNocalhostDevContainerPod()
		must(err)

		log.Infof("Syncthing port-forward pod %s, namespace %s", podName, nocalhostApp.NameSpace)

		svcProfile, _ := nocalhostSvc.GetProfile()
		// Start a pf for syncthing
		must(nocalhostSvc.PortForward(podName, svcProfile.RemoteSyncthingPort, svcProfile.RemoteSyncthingPort, "SYNC"))

		// TODO
		// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished),
		// the files that have been synchronized will not be synchronized.
		newSyncthing, err := nocalhostSvc.NewSyncthing(fileSyncOps.Container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin,
			fileSyncOps.SyncDouble)
		utils.ShouldI(err, "Failed to new syncthing")

		// starts up a local syncthing
		utils.ShouldI(newSyncthing.Run(context.TODO()), "Failed to run syncthing")

		must(nocalhostSvc.SetSyncingStatus(true))

		if fileSyncOps.Override {
			var i = 10
			for {
				time.Sleep(time.Second)

				i--
				// to force override the remote changing
				client := nocalhostSvc.NewSyncthingHttpClient()
				err = client.FolderOverride()
				if err == nil {
					log.Info("Force overriding workDir's remote changing")
					break
				}

				if i < 0 {
					log.WarnE(err, "Fail to overriding workDir's remote changing")
					break
				}
			}
		}
	},
}
