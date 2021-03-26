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
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
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

		if nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("Service \"%s\" is already in developing", deployment)
		}

		devStartOps.Kubeconfig = kubeConfig
		log.Info("Starting DevMode...")

		profileV2, _ := profile.NewAppProfileV2(nocalhostApp.NameSpace, nocalhostApp.Name, false)

		svcProfile := profileV2.FetchSvcProfileV2FromProfile(deployment)
		if svcProfile == nil {
			log.Fatal("Svc profile not found")
			return
		}
		//svcProfile := nocalhostApp.GetSvcProfileV2(deployment)
		if devStartOps.WorkDir != "" {
			svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).WorkDir = devStartOps.WorkDir
		}
		if devStartOps.DevImage != "" {
			svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).Image = devStartOps.DevImage
		}
		if len(devStartOps.LocalSyncDir) > 0 {
			svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = devStartOps.LocalSyncDir
		}
		//_ = nocalhostApp.SaveProfile()
		profileV2.Save()
		profileV2.CloseDb()

		newSyncthing, err := nocalhostApp.NewSyncthing(deployment, devStartOps.Container, devStartOps.LocalSyncDir, false)
		if err != nil {
			log.FatalE(err, "Failed to create syncthing process, please try again.")
		}

		// try install syncthing
		if newSyncthing != nil {
			var downloadVersion = Version

			// for debug only
			if devStartOps.SyncthingVersion != "" {
				downloadVersion = devStartOps.SyncthingVersion
			}

			_, err := syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, GitCommit).InstallIfNeeded()
			if err != nil {
				log.FatalE(err, "Failed to install syncthing, and no syncthing available locally in "+newSyncthing.BinPath+" please try again.")
			}
		}

		// set syncthing secret
		config, err := newSyncthing.GetRemoteConfigXML()
		syncSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: deployment + "-" + secret_config.SecretName,
				//Namespace: nocalhostApp.AppProfile.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"config.xml": config,
				"cert.pem":   []byte(secret_config.CertPEM),
				"key.pem":    []byte(secret_config.KeyPEM),
			},
		}
		err = nocalhostApp.CreateSyncThingSecret(deployment, syncSecret)
		if err != nil {
			log.Fatalf("Failed to create syncthing secret, please try to delete \"%s\" secret first manually.", syncthing.SyncSecretName)
		}

		// Stop port-forward
		appProfile, _ := nocalhostApp.GetProfile()
		pfList := appProfile.FetchSvcProfileV2FromProfile(deployment).DevPortForwardList
		for _, pf := range pfList {
			log.Infof("Stopping %d:%d", pf.LocalPort, pf.RemotePort)
			if err = nocalhostApp.EndDevPortForward(deployment, pf.LocalPort, pf.RemotePort); err != nil {
				log.WarnE(err, "")
			}
		}

		if err = nocalhostApp.ReplaceImage(context.TODO(), deployment, devStartOps); err != nil {
			log.WarnE(err, "Failed to replace dev container")
			log.Info("Resetting workload...")
			nocalhostApp.Reset(deployment)
			os.Exit(1)
		}

		if err = nocalhostApp.SetDevelopingStatus(deployment, true); err != nil {
			log.Fatal("Failed to update \"developing\" status\n")
		}

		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		if err != nil {
			log.FatalE(err, "")
		}

		for _, pf := range pfList {
			if err = nocalhostApp.PortForward(deployment, podName, pf.LocalPort, pf.RemotePort); err != nil {
				log.WarnE(err, "")
			}
		}

		if err = nocalhostApp.PortForwardAfterDevStart(deployment, devStartOps.Container, app.Deployment); err != nil {
			log.FatalE(err, "")
		}

	},
}
