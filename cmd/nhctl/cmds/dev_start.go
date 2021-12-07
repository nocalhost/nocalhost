/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"strconv"
	"strings"
)

var (
	deployment  string
	serviceType string
	pod         string
	shell       string
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
	devStartCmd.Flags().StringVarP(
		&devStartOps.DevImage, "image", "i", "",
		"image of DevContainer",
	)
	devStartCmd.Flags().StringVarP(
		&devStartOps.Container, "container", "c", "",
		"container to develop",
	)
	//devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work directory")
	devStartCmd.Flags().StringVar(&devStartOps.StorageClass, "storage-class", "", "StorageClass used by PV")
	devStartCmd.Flags().StringVar(
		&devStartOps.PriorityClass, "priority-class", "", "PriorityClass used by devContainer",
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
	devStartCmd.Flags().StringVarP(
		&shell, "shell", "", "",
		"use current shell cmd to enter terminal while dev start success",
	)
	devStartCmd.Flags().BoolVar(
		&devStartOps.NoTerminal, "without-terminal", false,
		"do not enter terminal directly while dev start success",
	)
	devStartCmd.Flags().BoolVar(
		&devStartOps.NoSyncthing, "without-sync", false,
		"do not start file-sync while dev start success",
	)
	devStartCmd.Flags().StringVarP(
		&devStartOps.DevModeType, "dev-mode", "m", "",
		"specify which DevMode you want to enter, such as: replace,duplicate. Default: replace",
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

		dt := profile.DevModeType(devStartOps.DevModeType)
		if !dt.IsDuplicateDevMode() && !dt.IsReplaceDevMode() {
			log.Fatalf("Unsupported DevModeType %s", dt)
		}

		if len(devStartOps.LocalSyncDir) > 1 {
			log.Fatal("Can not define multi 'local-sync(-s)'")
		} else if len(devStartOps.LocalSyncDir) == 0 {
			log.Fatal("'local-sync(-s)' must be specified")
		}

		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if !nocalhostApp.GetAppMeta().IsInstalled() {
			log.Fatal(nocalhostApp.GetAppMeta().NotInstallTips())
		}

		if nocalhostSvc.IsInDevMode() {
			coloredoutput.Hint(fmt.Sprintf("Already in %s DevMode...", nocalhostSvc.DevModeType.ToString()))

			podName, err := nocalhostSvc.BuildPodController().GetNocalhostDevContainerPod()
			must(err)

			if !devStartOps.NoSyncthing {
				if nocalhostSvc.IsProcessor() {
					startSyncthing(podName, devStartOps.Container, true)
				}
			} else {
				coloredoutput.Success("File sync is not resumed caused by --without-sync flag.")
			}

			if nocalhostSvc.IsProcessor() {
				if !devStartOps.NoTerminal || shell != "" {
					must(nocalhostSvc.EnterPodTerminal(podName, devStartOps.Container, shell))
				}
				return
			}
		}

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

		coloredoutput.Hint(fmt.Sprintf("Starting %s DevMode...", dt.ToString()))

		loadLocalOrCmConfigIfValid()
		stopPreviousSyncthing()
		recordLocalSyncDirToProfile()
		prepareSyncThing()
		stopPreviousPortForward()
		if err := enterDevMode(dt); err != nil {
			log.FatalE(err, "")
		}

		devPodName, err := nocalhostSvc.BuildPodController().GetNocalhostDevContainerPod()
		must(err)

		startPortForwardAfterDevStart(devPodName)

		if !devStartOps.NoSyncthing {
			startSyncthing(devPodName, devStartOps.Container, false)
		} else {
			coloredoutput.Success("File sync is not started caused by --without-sync flag..")
		}

		if !devStartOps.NoTerminal || shell != "" {
			must(nocalhostSvc.EnterPodTerminal(devPodName, "nocalhost-dev", shell))
		}

	},
}

var pfListBeforeDevStart []*profile.DevPortForward

func stopPreviousPortForward() {
	appProfile, _ := nocalhostApp.GetProfile()
	pfListBeforeDevStart = appProfile.SvcProfileV2(deployment, string(nocalhostSvc.Type)).DevPortForwardList
	for _, pf := range pfListBeforeDevStart {
		log.Infof("Stopping %d:%d", pf.LocalPort, pf.RemotePort)
		utils.Should(nocalhostSvc.EndDevPortForward(pf.LocalPort, pf.RemotePort))
	}
}

func prepareSyncThing() {
	dt := profile.DevModeType(devStartOps.DevModeType)
	must(nocalhostSvc.CreateSyncThingSecret(devStartOps.Container, devStartOps.LocalSyncDir, dt.IsDuplicateDevMode()))
}

func recordLocalSyncDirToProfile() {
	must(
		nocalhostSvc.UpdateSvcProfile(
			func(svcProfile *profile.SvcProfileV2) error {
				svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = devStartOps.LocalSyncDir
				return nil
			},
		),
	)
}

// when re enter dev mode, nocalhost will check the associate dir
// nocalhost will load svc config from associate dir if needed
func loadLocalOrCmConfigIfValid() {

	svcPack := dev_dir.NewSvcPack(
		nocalhostSvc.NameSpace,
		nocalhostSvc.AppName,
		nocalhostSvc.Type,
		nocalhostSvc.Name,
		devStartOps.Container,
	)

	switch len(devStartOps.LocalSyncDir) {
	case 0:
		associatePath := svcPack.GetAssociatePath()
		if associatePath == "" {
			must(errors.New("'local-sync(-s)' should specify while svc is not associate with local dir"))
		}
		devStartOps.LocalSyncDir = append(devStartOps.LocalSyncDir, string(associatePath))

		must(associatePath.Associate(svcPack, kubeConfig, true))

		_ = nocalhostApp.ReloadSvcCfg(deployment, base.SvcTypeOf(serviceType), false, false)
	case 1:

		must(dev_dir.DevPath(devStartOps.LocalSyncDir[0]).Associate(svcPack, kubeConfig, true))

		_ = nocalhostApp.ReloadSvcCfg(deployment, base.SvcTypeOf(serviceType), false, false)
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
			func(pid int) error {
				return syncthing.Stop(pid, true)
			},
		),
	)
	// kill syncthing process by find find it with terminal
	str := strings.ReplaceAll(nocalhostSvc.GetApplicationSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")
	utils2.KillSyncthingProcess(str)
}

func startSyncthing(podName, container string, resume bool) {
	StartSyncthing(podName, resume, false, container, nil, false)
	if resume {
		coloredoutput.Success("File sync resumed")
	} else {
		coloredoutput.Success("File sync started")
	}
}

func enterDevMode(devModeType profile.DevModeType) error {
	must(
		nocalhostSvc.AppMeta.SvcDevStarting(nocalhostSvc.Name, nocalhostSvc.Type,
			nocalhostApp.Identifier, devModeType),
	)
	must(nocalhostSvc.UpdateSvcProfile(func(v2 *profile.SvcProfileV2) error {
		//v2.DevModeType = devModeType
		v2.OriginDevContainer = devStartOps.Container
		return nil
	}))

	// prevent dev status modified but not actually enter dev mode
	var devStartSuccess = false
	var err error
	defer func() {
		if !devStartSuccess {
			log.Infof("Roll backing dev mode...")
			//if devModeType != "" {
			//	err = nocalhostSvc.UpdateSvcProfile(func(v2 *profile.SvcProfileV2) error {
			//		//v2.DevModeType = ""
			//		return nil
			//	})
			//	log.WarnE(err, "")
			//}
			_ = nocalhostSvc.AppMeta.SvcDevEnd(nocalhostSvc.Name, nocalhostSvc.Identifier, nocalhostSvc.Type, devModeType)
		}
	}()

	// Only `replace` DevMode needs to disable hpa
	if devModeType.IsReplaceDevMode() {
		log.Info("Disabling hpa...")
		hl, err := nocalhostSvc.ListHPA()
		if err != nil {
			log.WarnE(err, "Failed to find hpa")
		}
		if len(hl) == 0 {
			log.Info("No hpa found")
		}
		for _, h := range hl {
			if len(h.Annotations) == 0 {
				h.Annotations = make(map[string]string)
			}
			h.Annotations[_const.HPAOriginalMaxReplicasKey] = strconv.Itoa(int(h.Spec.MaxReplicas))
			h.Spec.MaxReplicas = 1
			if h.Spec.MinReplicas != nil {
				h.Annotations[_const.HPAOriginalMinReplicasKey] = strconv.Itoa(int(*h.Spec.MinReplicas))
				var i int32 = 1
				h.Spec.MinReplicas = &i
			}
			if _, err = nocalhostSvc.Client.UpdateHPA(&h); err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to update hpa %s", h.Name))
			} else {
				log.Infof("HPA %s has been disabled", h.Name)
			}
		}
	}

	nocalhostSvc.DevModeType = devModeType
	if err = nocalhostSvc.BuildPodController().ReplaceImage(context.TODO(), devStartOps); err != nil {
		log.WarnE(err, "Failed to replace dev container")
		log.Info("Resetting workload...")
		_ = nocalhostSvc.DevEnd(true)
		if errors.Is(err, nocalhost.CreatePvcFailed) {
			log.Info("Failed to provision persistent volume due to insufficient resources")
		}
		return err
	}

	if err = nocalhostSvc.AppMeta.SvcDevStartComplete(
		nocalhostSvc.Name, nocalhostSvc.Type, nocalhostSvc.Identifier, devModeType,
	); err != nil {
		return err
	}

	utils.Should(nocalhostSvc.IncreaseDevModeCount())

	// mark dev start as true
	devStartSuccess = true
	coloredoutput.Success("Dev container has been updated")
	return nil
}

func startPortForwardAfterDevStart(devPodName string) {
	for _, pf := range pfListBeforeDevStart {
		utils.Should(nocalhostSvc.PortForward(devPodName, pf.LocalPort, pf.RemotePort, pf.Role))
	}
	must(nocalhostSvc.PortForwardAfterDevStart(devPodName, devStartOps.Container))
}
