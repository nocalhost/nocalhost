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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

//type FileSyncFlags struct {
//	*EnvSettings
//	LocalSharedFolder string
//	RemoteDir      string
//	LocalSshPort           int
//	Deployment        string
//}
//
//var fileSyncFlags = &FileSyncFlags{
//	EnvSettings: settings,
//}

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(&fileSyncOps.LocalSharedFolder, "local-shared-folder", "l", "", "local folder to sync")
	fileSyncCmd.Flags().StringVarP(&fileSyncOps.RemoteDir, "remote-folder", "r", "", "remote folder path")
	fileSyncCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	fileSyncCmd.Flags().IntVarP(&fileSyncOps.LocalSshPort, "port", "p", 0, "local port which forwards to remote ssh port")
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
		if !nh.CheckIfApplicationExist(applicationName) {
			log.Fatalf("application \"%s\" not found\n", applicationName)
		}
		nocalhostApp, err = app.NewApplication(applicationName)
		//clientgoutils.Must(err)
		if err != nil {
			log.Fatalf("fail to get app info\n")
		}
		if deployment == "" {
			// todo record default deployment
			log.Fatal("please use -d to specify a k8s deployment")
		}
		err = nocalhostApp.FileSync(deployment, fileSyncOps)
		if err != nil {
			log.Fatal("fail to sync files")
		}
		err = nocalhostApp.SetSyncingStatus(deployment, true)
		if err != nil {
			log.Fatal("fail to update \"syncing\" status\n")
		}
	},
}
