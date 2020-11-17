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
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/third_party/mutagen"
)

type FileSyncFlags struct {
	*EnvSettings
	LocalSharedFolder string
	RemoteFolder      string
	SshPort           int
	Deployment        string
}

var fileSyncFlags = &FileSyncFlags{
	EnvSettings: settings,
}

func init() {
	fileSyncCmd.Flags().StringVarP(&fileSyncFlags.LocalSharedFolder, "local-shared-folder", "l", "", "local folder to sync")
	fileSyncCmd.Flags().StringVarP(&fileSyncFlags.RemoteFolder, "remote-folder", "r", "", "remote folder path")
	fileSyncCmd.Flags().StringVarP(&fileSyncFlags.Deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	fileSyncCmd.Flags().IntVarP(&fileSyncFlags.SshPort, "port", "p", 0, "local port which forwards to remote ssh port")
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
		nocalhostApp, err = nhctl.NewApplication(applicationName)
		clientgoutils.Must(err)
		if fileSyncFlags.Deployment == "" {
			// todo record default deployment
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}
		svcConfig := nocalhostApp.Config.GetSvcConfig(fileSyncFlags.Deployment)
		localDirsToSync := make([]string, 0)
		if fileSyncFlags.LocalSharedFolder == "" {
			// reading from config
			if svcConfig != nil && svcConfig.Sync != nil && len(svcConfig.Sync) > 0 {
				debug("[nocalhost config] reading local shared folder config ...")
				//fileSyncFlags.LocalSharedFolder = svcConfig.LocalWorkDir
				for _, dir := range svcConfig.Sync {
					localDirsToSync = append(localDirsToSync, dir)
				}
			} else {
				fmt.Println("error: please use -l flag or set localSharedFolder config to specify a local directory to sync with remote")
				return
			}
		} else {
			localDirsToSync = append(localDirsToSync, fileSyncFlags.LocalSharedFolder)
		}

		if fileSyncFlags.RemoteFolder == "" {
			if svcConfig != nil && svcConfig.WorkDir != "" {
				debug("[nocalhost config] reading mountPath config ...")
				fileSyncFlags.RemoteFolder = svcConfig.WorkDir
			} else {
				fmt.Println("error: please use -r flag or set mountPath config to specify a remote folder")
				return
			}
		}

		if fileSyncFlags.SshPort == 0 {
			if svcConfig != nil && svcConfig.SshPort != nil {
				if svcConfig.SshPort.LocalPort != 0 {
					fileSyncFlags.SshPort = svcConfig.SshPort.LocalPort
				} else {
					fmt.Println("fail to get ssh port, it may be a todo item")
					return
				}
			}
		}
		fmt.Println("file syncing...") // tools/darwin/mutagen sync create --sync-mode=one-way-safe --releaseName=$1  $2  $3
		// ./tools/script/file-sync.sh coding dir01 root@127.0.0.1:12345:/home/code
		for _, dir := range localDirsToSync {
			fmt.Printf("syncing %s ...\n", dir)
			mutagen.FileSync(dir, fileSyncFlags.RemoteFolder, fmt.Sprintf("%d", fileSyncFlags.SshPort))
		}
	},
}
