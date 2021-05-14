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
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
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
	serviceType string
	pod         string
)

var devStartOps = &model.DevStartOptions{}

func init() {

	devStartCmd.Flags().StringVarP(
		&deployment, "deployment", "d", "",
		"k8s deployment your developing service exists",
	)
	devStartCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	devStartCmd.Flags().StringVarP(&devStartOps.DevImage, "image", "i", "", "image of DevContainer")
	devStartCmd.Flags().StringVarP(&devStartOps.Container, "container", "c", "", "container to develop")
	devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work directory")
	devStartCmd.Flags().StringVar(&devStartOps.StorageClass, "storage-class", "", "StorageClass used by PV")
	devStartCmd.Flags().StringVar(
		&devStartOps.PriorityClass, "priority-class", "", "PriorityClass used by devContainer",
	)
	devStartCmd.Flags().StringVar(
		&devStartOps.SideCarImage, "sidecar-image", "",
		"image of nocalhost-sidecar container",
	)
	// for debug only
	devStartCmd.Flags().StringVar(
		&devStartOps.SyncthingVersion, "syncthing-version", "",
		"versions of syncthing and this flag is use for debug only",
	)
	// local absolute paths to sync
	devStartCmd.Flags().StringSliceVarP(
		&devStartOps.LocalSyncDir, "local-sync", "s", []string{},
		"local directory to sync",
	)
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
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if !nocalhostApp.GetAppMeta().IsInstalled() {
			log.Fatal(nocalhostApp.GetAppMeta().NotInstallTips())
		}

		if len(devStartOps.LocalSyncDir) == 1 {
			must(nocalhostSvc.Associate(devStartOps.LocalSyncDir[0]))
		} else {
			log.Fatal(errors.New("Can not define multi 'local-sync(-s)'"))
		}

		nocalhostApp.LoadSvcCfgFromLocalIfNeeded(deployment, serviceType, false)

		devStartOps.Kubeconfig = kubeConfig
		log.Info("Starting DevMode...")

		// Clean up previous syncthing
		must(
			nocalhostSvc.FindOutSyncthingProcess(
				func(pid int, pidFile string) error {
					return syncthing.Stop(pid, "", false)
				},
			),
		)

		must(
			nocalhostSvc.UpdateProfile(
				func(p *profile.AppProfileV2, svcProfile *profile.SvcProfileV2) error {
					if svcProfile == nil {
						return errors.New("Svc profile not found")
					}
					if devStartOps.WorkDir != "" {
						svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).WorkDir = devStartOps.WorkDir
					}
					if devStartOps.DevImage != "" {
						svcProfile.GetContainerDevConfigOrDefault(devStartOps.Container).Image = devStartOps.DevImage
					}
					if len(devStartOps.LocalSyncDir) == 1 {
						svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = devStartOps.LocalSyncDir
					} else {
						return errors.New("Can not define multi 'local-sync(-s)'")
					}

					p.GenerateIdentifierIfNeeded()
					return nil
				},
			),
		)

		p, err := nocalhostApp.GetProfile()
		must(err)

		must(
			nocalhostSvc.AppMeta.SvcDevStart(
				nocalhostSvc.Name, nocalhostSvc.Type, p.Identifier,
			),
		)

		// prevent dev status modified but not actually enter dev mode
		var devStartSuccess = false
		defer func() {
			if !devStartSuccess {
				_ = nocalhostSvc.AppMeta.SvcDevEnd(nocalhostSvc.Name, nocalhostSvc.Type)
			}
		}()

		newSyncthing, err := nocalhostSvc.NewSyncthing(devStartOps.Container, devStartOps.LocalSyncDir, false)
		mustI(err, "Failed to create syncthing process, please try again")

		// try install syncthing
		var downloadVersion = Version

		// for debug only
		if devStartOps.SyncthingVersion != "" {
			downloadVersion = devStartOps.SyncthingVersion
		}

		_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, GitCommit).InstallIfNeeded()
		mustI(
			err, "Failed to install syncthing, no syncthing available locally in "+
				newSyncthing.BinPath+" please try again.",
		)

		// set syncthing secret
		config, err := newSyncthing.GetRemoteConfigXML()
		must(err)

		syncSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: nocalhostSvc.GetSyncThingSecretName(),
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"config.xml": config,
				"cert.pem":   []byte(secret_config.CertPEM),
				"key.pem":    []byte(secret_config.KeyPEM),
			},
		}
		must(nocalhostSvc.CreateSyncThingSecret(syncSecret))

		// Stop port-forward
		appProfile, _ := nocalhostApp.GetProfile()
		pfList := appProfile.SvcProfileV2(deployment, string(nocalhostSvc.Type)).DevPortForwardList
		for _, pf := range pfList {
			log.Infof("Stopping %d:%d", pf.LocalPort, pf.RemotePort)
			utils.Should(nocalhostSvc.EndDevPortForward(pf.LocalPort, pf.RemotePort))
		}

		if err = nocalhostSvc.BuildPodController().ReplaceImage(context.TODO(), devStartOps); err != nil {
			log.WarnE(err, "Failed to replace dev container")
			log.Info("Resetting workload...")
			_ = nocalhostSvc.DevEnd(true)
			if errors.Is(err, nocalhost.CreatePvcFailed) {
				log.Info("Create PVC failed, please make sure you have the required PVC quota")
			}
			os.Exit(1)
		}

		podName, err := nocalhostSvc.GetNocalhostDevContainerPod()
		must(err)

		// mark dev start as true
		devStartSuccess = true
		for _, pf := range pfList {
			utils.Should(nocalhostSvc.PortForward(podName, pf.LocalPort, pf.RemotePort, pf.Role))
		}
		must(nocalhostSvc.PortForwardAfterDevStart(devStartOps.Container))
	},
}
