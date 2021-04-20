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
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deployment  string
	ServiceType string
	//Container   string
	pod string
)

var devStartOps = &app.DevStartOptions{}

func init() {

	devStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&devStartOps.DevImage, "image", "i", "", "image of DevContainer")
	devStartCmd.Flags().StringVarP(&devStartOps.Container, "container", "c", "", "container to develop")
	devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work directory, same as sync path")
	devStartCmd.Flags().StringVar(&devStartOps.StorageClass, "storage-class", "", "the StorageClass used by persistent volumes")
	devStartCmd.Flags().StringVar(&devStartOps.PriorityClass, "priority-class", "", "the PriorityClass used by devContainer")
	devStartCmd.Flags().StringVar(&devStartOps.SideCarImage, "sidecar-image", "", "image of nocalhost-sidecar container")

	// for debug only
	devStartCmd.Flags().StringVar(&devStartOps.SyncthingVersion, "syncthing-version", "", "versions of syncthing and this flag is use for debug only")
	// LocalSyncDir is local absolute path to sync provided by plugin
	devStartCmd.Flags().StringSliceVarP(&devStartOps.LocalSyncDir, "local-sync", "s", []string{}, "local directory to sync")
	debugCmd.AddCommand(devStartCmd)
}

var devStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "Start DevMode",
	Long:  `Start DevMode`,
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

		if !nocalhostApp.GetAppMeta().IsInstalled() {
			log.Fatalf(nocalhostApp.GetAppMeta().NotInstallTips())
		}

		devStartOps.Kubeconfig = kubeConfig
		log.Info("Starting DevMode...")

		profileV2, err := profile.NewAppProfileV2ForUpdate(nocalhostApp.NameSpace, nocalhostApp.Name)
		must(err)

		svcProfile := profileV2.FetchSvcProfileV2FromProfile(deployment)
		if svcProfile == nil {
			log.Fatal("Svc profile not found")
			return
		}
		if devStartOps.WorkDir != "" {
			svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).WorkDir = devStartOps.WorkDir
		}
		if devStartOps.DevImage != "" {
			svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).Image = devStartOps.DevImage
		}
		if len(devStartOps.LocalSyncDir) > 0 {
			svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = devStartOps.LocalSyncDir
		}
		profileV2.GenerateIdentifierIfNeeded()

		profileV2.Save()
		profileV2.CloseDb()

		must(nocalhostApp.GetAppMeta().DeploymentDevStart(deployment, profileV2.Identifier))

		newSyncthing, err := nocalhostApp.NewSyncthing(deployment, devStartOps.Container, devStartOps.LocalSyncDir, false)
		mustI(err, "Failed to create syncthing process, please try again")

		// try install syncthing
		var downloadVersion = Version

		// for debug only
		if devStartOps.SyncthingVersion != "" {
			downloadVersion = devStartOps.SyncthingVersion
		}

		_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, GitCommit).InstallIfNeeded()
		mustI(err, "Failed to install syncthing, and no syncthing available locally in "+newSyncthing.BinPath+" please try again.")

		// set syncthing secret
		config, err := newSyncthing.GetRemoteConfigXML()
		syncSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: deployment + "-" + secret_config.SecretName,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"config.xml": config,
				"cert.pem":   []byte(secret_config.CertPEM),
				"key.pem":    []byte(secret_config.KeyPEM),
			},
		}
		must(nocalhostApp.CreateSyncThingSecret(deployment, syncSecret))

		// Stop port-forward
		appProfile, _ := nocalhostApp.GetProfile()
		pfList := appProfile.FetchSvcProfileV2FromProfile(deployment).DevPortForwardList
		for _, pf := range pfList {
			log.Infof("Stopping %d:%d", pf.LocalPort, pf.RemotePort)
			utils.Should(nocalhostApp.EndDevPortForward(deployment, pf.LocalPort, pf.RemotePort))
		}

		if err = nocalhostApp.ReplaceImage(context.TODO(), deployment, devStartOps); err != nil {
			log.WarnE(err, "Failed to replace dev container")
			log.Info("Resetting workload...")
			_ = nocalhostApp.DevEnd(deployment, true)
			os.Exit(1)
		}
		must(nocalhostApp.SetDevelopingStatus(deployment, true))

		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		must(err)

		for _, pf := range pfList {
			utils.Should(nocalhostApp.PortForward(deployment, podName, pf.LocalPort, pf.RemotePort, pf.Role))
		}

		must(nocalhostApp.PortForwardAfterDevStart(deployment, devStartOps.Container, app.Deployment))
	},
}
