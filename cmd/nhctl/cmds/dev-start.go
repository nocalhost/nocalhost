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
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/secret"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/pkg/nhctl/log"
)

var (
	nameSpace  string
	deployment string
)

var devStartOps = &app.DevStartOptions{}

func init() {

	devStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&devStartOps.DevLang, "lang", "l", "", "the development language, eg: java go python")
	devStartCmd.Flags().StringVarP(&devStartOps.DevImage, "image", "i", "", "image of development container")
	devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work directory, same as sync path")
	devStartCmd.Flags().StringVar(&devStartOps.SideCarImage, "sidecar-image", "", "image of sidecar container")
	// LocalSyncDir is local sync directory Absolute path splice by plugin
	devStartCmd.Flags().StringSliceVarP(&devStartOps.LocalSyncDir, "local-sync", "s", []string{}, "local sync directory")
	debugCmd.AddCommand(devStartCmd)
}

var devStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "enter dev model",
	Long:  `enter dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if settings.Debug {
			log.SetLevel(logrus.DebugLevel)
		}
		applicationName := args[0]
		InitAppAndSvc(applicationName, deployment)

		if nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" is already developing", deployment)
		}

		nocalhostApp.LoadOrCreateSvcProfile(deployment, app.Deployment)

		// set develop status first, avoid stack in dev start and break, or it will never resume
		err = nocalhostApp.SetDevelopingStatus(deployment, true)
		if err != nil {
			log.Fatal("fail to update \"developing\" status\n")
		}

		devStartOps.Kubeconfig = settings.KubeConfig
		fmt.Println("entering development model...")

		// set dev start ops args
		// devStartOps.LocalSyncDir is from pulgin by local-sync
		var fileSyncOptions = &app.FileSyncOptions{}
		devStartOps.Namespace = nocalhostApp.AppProfile.Namespace
		if devStartOps.WorkDir == "" { // command not pass this arguments
			devStartOps.WorkDir = nocalhostApp.GetDefaultWorkDir(deployment)
		}
		nocalhostApp.SetSvcWorkDir(deployment, devStartOps.WorkDir)
		// syncthings
		newSyncthing, err := syncthing.New(nocalhostApp, deployment, devStartOps, fileSyncOptions)
		if err != nil {
			log.Fatalf("create syncthing fail, please try again.")
		}
		// install syncthing
		if newSyncthing != nil && !newSyncthing.IsInstalled() {
			err = newSyncthing.DownloadSyncthing()
			if err != nil {
				log.Fatalf("download syncthing fail, please try again")
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
		err = nocalhostApp.CreateSyncThingSecret(syncSecret, devStartOps)
		if err != nil {
			// TODO dev end should delete syncthing secret
			log.Fatalf("create syncthing secret fail, please try to manual delete %s secret first", syncthing.SyncSecretName)
		}
		// set profile sync dir
		err = nocalhostApp.SetLocalAbsoluteSyncDirFromDevStartPlugin(deployment, devStartOps.LocalSyncDir)
		if err != nil {
			log.Fatalf("fail to update sync directory")
		}
		// end syncthing doing
		err = nocalhostApp.ReplaceImage(context.TODO(), deployment, devStartOps)
		if err != nil {
			log.Fatalf("fail to replace dev container: err%v\n", err)
		}
		// set profile sync port
		err = nocalhostApp.SetSyncthingPort(deployment, newSyncthing.RemotePort, newSyncthing.RemoteGUIPort, newSyncthing.LocalPort, newSyncthing.LocalGUIPort)
		if err != nil {
			log.Fatal("fail to update \"developing\" syncthing port status\n")
		}
	},
}
