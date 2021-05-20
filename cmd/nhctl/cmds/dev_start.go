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
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
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
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if !nocalhostApp.GetAppMeta().IsInstalled() {
			log.Fatal(nocalhostApp.GetAppMeta().NotInstallTips())
		}

		if nocalhostApp.GetAppMeta().CheckIfSvcDeveloping(nocalhostSvc.Name, nocalhostSvc.Type) {
			coloredoutput.Hint("Already in DevMode... entering container")

			podName, err := nocalhostSvc.GetNocalhostDevContainerPod()
			mustP(err)

			startSyncthing(true)
			must(nocalhostSvc.EnterPodTerminal(podName, container))
		} else {

			// 1) reload svc config from local if needed
			// 2) stop previous syncthing
			// 3) recording profile
			// 4) mark app meta as developing
			// 5) initial syncthing runtime env
			// 6) stop port-forward
			// 7) enter developing (replace image)
			// 8) port forward for dev-container
			// 9) start syncthing
			// 10) entering dev container

			coloredoutput.Hint("Starting DevMode...")

			loadLocalConfigIfNeeded()
			stopPreviousSyncthing()
			recordingProfile()
			podName := enterDevMode()
			startSyncthing(false)
			must(nocalhostSvc.EnterPodTerminal(podName, container))
		}
	},
}

func recordingProfile() {
	must(
		nocalhostSvc.UpdateProfile(
			func(p *profile.AppProfileV2, svcProfile *profile.SvcProfileV2) error {
				if svcProfile == nil {
					return errors.New(
						fmt.Sprintf(
							"Svc Profile not found %s-%s-%s", p.Namespace, nocalhostSvc.Type, nocalhostSvc.Name,
						),
					)
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
}

// when re enter dev mode, nocalhost will check the associate dir
// nocalhost will load svc config from associate dir if needed
func loadLocalConfigIfNeeded() {

	switch len(devStartOps.LocalSyncDir) {
	case 0:
		p, err := nocalhostSvc.GetProfile()
		must(err)

		if p.Associate == "" {
			must(errors.New("'local-sync(-s)' should specify while svc is not associate with local dir"))
		}
		devStartOps.LocalSyncDir = append(devStartOps.LocalSyncDir, p.Associate)

		nocalhostApp.LoadSvcCfgFromLocalIfNeeded(deployment, serviceType, false)
	case 1:
		must(nocalhostSvc.Associate(devStartOps.LocalSyncDir[0]))
		nocalhostApp.LoadSvcCfgFromLocalIfNeeded(deployment, serviceType, false)
	default:
		log.Fatal(errors.New("Can not define multi 'local-sync(-s)'"))
	}
}

// we should clean previous Syncthing
// prevent previous syncthing hold the db lock
func stopPreviousSyncthing() {
	// Clean up previous syncthing
	must(
		nocalhostSvc.FindOutSyncthingProcess(
			func(pid int, pidFile string) error {
				return syncthing.Stop(pid, "", false)
			},
		),
	)
}

func startSyncthing(resume bool) {
	nhctl, err := utils.GetNhctlPath()
	must(err)

	var params = []string{
		"sync", nocalhostApp.Name, "-d", nocalhostSvc.Name, "-t", nocalhostSvc.Type.String(),
		"--kubeconfig", kubeConfig, "-n", nameSpace,
	}

	if resume {
		params = append(params, "--resume")
		fmt.Println()
		coloredoutput.Success("File sync resuming")
	}else {
		fmt.Println()
		coloredoutput.Success("File sync starting")
	}

	_, err = tools.ExecCommand(
		nil, true, true, false,
		nhctl, params...,
	)
	must(err)
}

func enterDevMode() string {
	must(
		nocalhostSvc.AppMeta.SvcDevStart(
			nocalhostSvc.Name, nocalhostSvc.Type, nocalhostApp.GetProfileCompel().Identifier,
		),
	)

	// prevent dev status modified but not actually enter dev mode
	var devStartSuccess = false
	defer func() {
		if !devStartSuccess {
			log.Infof("Roll backing dev mode... \n")
			_ = nocalhostSvc.AppMeta.SvcDevEnd(nocalhostSvc.Name, nocalhostSvc.Type)
		}

		if r := recover(); r != nil {
			os.Exit(1)
		}
	}()

	newSyncthing, err := nocalhostSvc.NewSyncthing(devStartOps.Container, devStartOps.LocalSyncDir, false)
	mustPI(err, "Failed to create syncthing process, please try again")

	// try install syncthing
	var downloadVersion = Version

	// for debug only
	if devStartOps.SyncthingVersion != "" {
		downloadVersion = devStartOps.SyncthingVersion
	}

	_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, GitCommit).InstallIfNeeded()
	mustPI(
		err, "Failed to install syncthing, no syncthing available locally in "+
			newSyncthing.BinPath+" please try again.",
	)

	// set syncthing secret
	config, err := newSyncthing.GetRemoteConfigXML()
	mustP(err)

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
	mustP(nocalhostSvc.CreateSyncThingSecret(syncSecret))

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
			log.Info("Failed to provision persistent volume due to insufficient resources")
		}
		mustP(err)
	}

	// mark dev start as true
	devStartSuccess = true

	podName, err := nocalhostSvc.GetNocalhostDevContainerPod()
	mustP(err)

	for _, pf := range pfList {
		utils.Should(nocalhostSvc.PortForward(podName, pf.LocalPort, pf.RemotePort, pf.Role))
	}
	mustP(nocalhostSvc.PortForwardAfterDevStart(devStartOps.Container))

	fmt.Println()
	coloredoutput.Success("Dev container has been updated")
	fmt.Println()

	return podName
}
