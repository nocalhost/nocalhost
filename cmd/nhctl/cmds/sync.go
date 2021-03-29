/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"context"
	"nocalhost/internal/nhctl/app"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.RunAsDaemon, "daemon", "m", true, "if file sync run as daemon")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.SyncDouble, "double", "b", false, "if use double side sync")
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Resume, "resume", false, "resume file sync, this will restart port-forward and syncthing")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.SyncedPattern, "synced-pattern", "s", []string{}, "local synced pattern")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.IgnoredPattern, "ignored-pattern", "i", []string{}, "local ignored pattern")
	fileSyncCmd.Flags().StringVar(&fileSyncOps.Container, "container", "", "container name of pod to sync")
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Override, "overwrite", true, "override the remote changing according to the local sync folder while start up")
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
		var err error
		applicationName := args[0]

		initAppAndCheckIfSvcExist(applicationName, deployment, nil)

		if b, _ := nocalhostApp.CheckIfSvcIsDeveloping(deployment); !b {
			log.Fatalf("Service \"%s\" is not in developing", deployment)
		}

		// resume port-forward and syncthing
		id, err := strconv.Atoi(os.Getenv(daemon.MARK_ENV_NAME))
		if err != nil || id == 0 {
			// run once in father progress
			if fileSyncOps.Resume {
				err = nocalhostApp.StopFileSyncOnly(deployment)
				if err != nil {
					log.WarnE(err, "Error occurs when stopping sync process, ignore")
				}
			}
		}

		//if nocalhostApp.CheckIfSvcIsSyncthing(deployment) {
		//	log.Fatalf("Service \"%s\" is already in syncing", deployment)
		//}

		// syncthing port-forward
		// set abs directory to call myself
		//nhctlAbsDir, err := exec.LookPath(utils.GetNhctlBinName())
		//if err != nil {
		//	log.Fatalf("Failed to load nhctl in %v", err)
		//}

		// overwrite Args[0] as ABS directory of bin directory
		//os.Args[0] = nhctlAbsDir

		// run in background
		//if fileSyncOps.RunAsDaemon {
		//	// when child progress run here, it will check env value and exit, so child will not run above code
		//	_, err := daemon.Background(nocalhostApp.GetPortSyncLogFile(deployment), nocalhostApp.GetApplicationBackGroundPortForwardPidFile(deployment), true)
		//	if err != nil {
		//		log.Fatalf("run port-forward background fail, please try again")
		//	}
		//}

		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		if err != nil {
			log.FatalE(err, "No dev container found")
		}

		log.Infof("Syncthing port-forward pod %s, namespace %s", podName, nocalhostApp.GetNamespace())

		appProfile, _ := nocalhostApp.GetProfile()
		svcProfile := appProfile.FetchSvcProfileV2FromProfile(deployment)
		// Start a pf for syncthing
		err = nocalhostApp.PortForward(svcProfile.ActualName, podName, svcProfile.RemoteSyncthingPort, svcProfile.RemoteSyncthingPort, "SYNC")
		if err != nil {
			log.FatalE(err, "")
		}

		// Getting pattern from svc profile first
		//appProfile, _ = nocalhostApp.GetProfile()
		//profile := appProfile.FetchSvcProfileV2FromProfile(deployment)
		if svcProfile.GetContainerDevConfigOrDefault(fileSyncOps.Container).Sync == nil {
			svcProfile.GetContainerDevConfigOrDefault(fileSyncOps.Container).Sync = &profile2.SyncConfig{}
		}
		if len(fileSyncOps.IgnoredPattern) != 0 {
			svcProfile.GetContainerDevConfigOrDefault(fileSyncOps.Container).Sync.IgnoreFilePattern = fileSyncOps.IgnoredPattern
		}
		if len(fileSyncOps.SyncedPattern) != 0 {
			svcProfile.GetContainerDevConfigOrDefault(fileSyncOps.Container).Sync.FilePattern = fileSyncOps.SyncedPattern
		}

		// TODO
		// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished), the files that have been synchronized will not be synchronized.
		newSyncthing, err := nocalhostApp.NewSyncthing(deployment, fileSyncOps.Container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin, fileSyncOps.SyncDouble)
		if err != nil {
			log.WarnE(err, "Failed to new syncthing")
		}

		// starts up a local syncthing
		err = newSyncthing.Run(context.TODO())
		if err != nil {
			log.WarnE(err, "Failed to run syncthing")
		}

		// set sync status in child progress
		err = nocalhostApp.SetSyncingStatus(deployment, true)
		if err != nil {
			log.Fatal("Failed to update syncing status")
		}

		if fileSyncOps.Override {
			var i = 10
			for {
				time.Sleep(time.Second)

				i--
				// to force override the remote changing
				client := nocalhostApp.NewSyncthingHttpClient(deployment)

				err = client.FolderOverride()
				if err == nil {
					log.Info("Force overriding workDir's remote changing")
					break
				}

				if i < 0 {
					log.ErrorE(err, "Fail to overriding workDir's remote changing")
					break
				}
			}
		}
	},
}
