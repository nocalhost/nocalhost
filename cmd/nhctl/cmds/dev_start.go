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
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/secret"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	nameSpace  string
	deployment string
)

var devStartOps = &app.DevStartOptions{}

func init() {

	devStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&devStartOps.DevLang, "lang", "l", "", "the program language, eg: java go python")
	devStartCmd.Flags().StringVarP(&devStartOps.DevImage, "image", "i", "", "image of DevContainer")
	devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work directory, same as sync path")
	devStartCmd.Flags().StringVar(&devStartOps.StorageClass, "storage-class", "", "the StorageClass used by persistent volumes")
	devStartCmd.Flags().StringVar(&devStartOps.SideCarImage, "sidecar-image", "", "image of nocalhost-sidecar container")

	// for debug only
	devStartCmd.Flags().StringVar(&devStartOps.SyncthingVersion, "syncthing-version", syncthing.SyncthingVersion, "versions of syncthing and this flag is use for debug only")
	// LocalSyncDir is local sync directory Absolute path splice by plugin
	devStartCmd.Flags().StringSliceVarP(&devStartOps.LocalSyncDir, "local-sync", "s", []string{}, "local sync directory")
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
		InitAppAndCheckIfSvcExist(applicationName, deployment)

		if nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" is already in developing", deployment)
		}

		//nocalhostApp.LoadConfigToSvcProfile(deployment, app.Deployment)

		devStartOps.Kubeconfig = settings.KubeConfig
		log.Info("starting DevMode...")

		// set dev start ops args
		// devStartOps.LocalSyncDir is from plugin by local-sync
		var fileSyncOptions = &app.FileSyncOptions{}
		devStartOps.Namespace = nocalhostApp.AppProfile.Namespace
		if devStartOps.WorkDir == "" { // command flag not set
			devStartOps.WorkDir = nocalhostApp.GetDefaultWorkDir(deployment)
		}
		nocalhostApp.SetSvcWorkDir(deployment, devStartOps.WorkDir)

		newSyncthing, err := nocalhostApp.NewSyncthing(deployment, devStartOps, fileSyncOptions)
		if err != nil {
			log.FatalE(err, "failed to create syncthing process, please try again.")
		}
		// install syncthing
		if newSyncthing != nil && !newSyncthing.IsInstalled() {
			err = newSyncthing.DownloadSyncthing(devStartOps.SyncthingVersion)
			if err != nil {
				log.FatalE(err, "failed to download syncthing binary, please try again.")
			}
		}

		// set syncthing secret
		config, err := secret.GetConfigXML(newSyncthing)
		syncSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment + "-" + secret_config.SecretName,
				Namespace: nocalhostApp.AppProfile.Namespace,
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
			// TODO dev end should delete syncthing secret
			log.Fatalf("failed to create syncthing secret, please try to delete \"%s\" secret first manually.", syncthing.SyncSecretName)
		}

		// set profile sync dir
		err = nocalhostApp.SetLocalAbsoluteSyncDirFromDevStartPlugin(deployment, devStartOps.LocalSyncDir)
		if err != nil {
			log.Fatalf("failed to update sync directory")
		}

		err = nocalhostApp.ReplaceImage(context.TODO(), deployment, devStartOps)
		if err != nil {
			// todo: rollback somethings
			log.FatalE(err, "failed to replace dev container")
		}
		// set profile sync port
		err = nocalhostApp.SetSyncthingPort(deployment, newSyncthing.RemotePort, newSyncthing.RemoteGUIPort, newSyncthing.LocalPort, newSyncthing.LocalGUIPort)
		if err != nil {
			log.Fatal("failed to update \"developing\" syncthing port status\n")
		}

		// TODO set develop status, avoid stack in dev start and break, or it will never resume
		err = nocalhostApp.SetDevelopingStatus(deployment, true)
		if err != nil {
			log.Fatal("failed to update \"developing\" status\n")
		}
	},
}
