/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dev

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/cmd/nhctl/cmds/common"
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
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"strconv"
	"strings"
)

var (
	pod   string
	shell string
)

var DevStartOps = &model.DevStartOptions{}

func init() {

	DevStartCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", "",
		"k8s deployment your developing service exists",
	)
	DevStartCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	DevStartCmd.Flags().StringVarP(
		&DevStartOps.DevImage, "image", "i", "",
		"image of DevContainer",
	)
	DevStartCmd.Flags().StringVarP(
		&DevStartOps.Container, "container", "c", "",
		"container to develop",
	)
	//DevStartCmd.Flags().StringVar(&DevStartOps.WorkDir, "work-dir", "", "container's work directory")
	DevStartCmd.Flags().StringVar(&DevStartOps.StorageClass, "storage-class", "", "StorageClass used by PV")
	DevStartCmd.Flags().StringVar(
		&DevStartOps.PriorityClass, "priority-class", "", "PriorityClass used by devContainer",
	)
	// for debug only
	DevStartCmd.Flags().StringVar(
		&DevStartOps.SyncthingVersion, "syncthing-version", "",
		"versions of syncthing and this flag is use for debug only",
	)
	// local absolute paths to sync
	DevStartCmd.Flags().StringSliceVarP(
		&DevStartOps.LocalSyncDir, "local-sync", "s", []string{},
		"local directory to sync",
	)
	DevStartCmd.Flags().StringVarP(
		&shell, "shell", "", "",
		"use current shell cmd to enter terminal while dev start success",
	)
	DevStartCmd.Flags().BoolVar(
		&DevStartOps.NoTerminal, "without-terminal", false,
		"do not enter terminal directly while dev start success",
	)
	DevStartCmd.Flags().BoolVar(
		&DevStartOps.NoSyncthing, "without-sync", false,
		"do not start file-sync while dev start success",
	)
	DevStartCmd.Flags().StringVarP(
		&DevStartOps.DevModeType, "dev-mode", "m", "",
		"specify which DevMode you want to enter, such as: replace,duplicate. Default: replace",
	)
}

var DevStartCmd = &cobra.Command{
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
		StartDevMode(args[0])
	},
}

func StartDevMode(applicationName string) {
	dt := profile.DevModeType(DevStartOps.DevModeType)
	if !dt.IsDuplicateDevMode() && !dt.IsReplaceDevMode() {
		log.Fatalf("Unsupported DevModeType %s", dt)
	}

	if len(DevStartOps.LocalSyncDir) > 1 {
		log.Fatal("Can not define multi 'local-sync(-s)'")
	} else if len(DevStartOps.LocalSyncDir) == 0 {
		log.Fatal("'local-sync(-s)' must be specified")
	}

	common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)

	if !common.NocalhostApp.GetAppMeta().IsInstalled() {
		log.Fatal(common.NocalhostApp.GetAppMeta().NotInstallTips())
	}

	if common.NocalhostSvc.IsInDevMode() {
		coloredoutput.Hint(fmt.Sprintf("Already in %s DevMode...", common.NocalhostSvc.DevModeType.ToString()))

		podName, err := common.NocalhostSvc.GetDevModePodName()
		must(err)

		if !DevStartOps.NoSyncthing {
			if common.NocalhostSvc.IsProcessor() {
				startSyncthing(podName, DevStartOps.Container, true)
			}
		} else {
			coloredoutput.Success("File sync is not resumed caused by --without-sync flag.")
		}

		if common.NocalhostSvc.IsProcessor() {
			if !DevStartOps.NoTerminal || shell != "" {
				must(common.NocalhostSvc.EnterPodTerminal(podName, DevStartOps.Container, shell))
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

	common.NocalhostSvc.DevModeType = dt
	loadLocalOrCmConfigIfValid()
	stopPreviousSyncthing()
	recordLocalSyncDirToProfile()
	prepareSyncThing()
	stopPreviousPortForward()
	if err := enterDevMode(dt); err != nil {
		log.FatalE(err, "")
	}

	devPodName, err := common.NocalhostSvc.GetDevModePodName()
	must(err)

	startPortForwardAfterDevStart(devPodName)

	if !DevStartOps.NoSyncthing {
		startSyncthing(devPodName, DevStartOps.Container, false)
	} else {
		coloredoutput.Success("File sync is not started caused by --without-sync flag..")
	}

	if !DevStartOps.NoTerminal || shell != "" {
		must(common.NocalhostSvc.EnterPodTerminal(devPodName, "nocalhost-dev", shell))
	}
}

var pfListBeforeDevStart []*profile.DevPortForward

func stopPreviousPortForward() {
	appProfile, _ := common.NocalhostApp.GetProfile()
	pfListBeforeDevStart = appProfile.SvcProfileV2(common.WorkloadName, string(common.NocalhostSvc.Type)).DevPortForwardList
	for _, pf := range pfListBeforeDevStart {
		log.Infof("Stopping %d:%d", pf.LocalPort, pf.RemotePort)
		utils.Should(common.NocalhostSvc.EndDevPortForward(pf.LocalPort, pf.RemotePort))
	}
}

func prepareSyncThing() {
	dt := profile.DevModeType(DevStartOps.DevModeType)
	clientgoutils.Must(common.NocalhostSvc.CreateSyncThingSecret(DevStartOps.Container, DevStartOps.LocalSyncDir, dt.IsDuplicateDevMode()))
}

func recordLocalSyncDirToProfile() {
	must(
		common.NocalhostSvc.UpdateSvcProfile(
			func(svcProfile *profile.SvcProfileV2) error {
				svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = DevStartOps.LocalSyncDir
				return nil
			},
		),
	)
}

// when re enter dev mode, nocalhost will check the associate dir
// nocalhost will load svc config from associate dir if needed
func loadLocalOrCmConfigIfValid() {

	svcPack := dev_dir.NewSvcPack(
		common.NocalhostSvc.NameSpace,
		common.NocalhostSvc.AppName,
		common.NocalhostSvc.Type,
		common.NocalhostSvc.Name,
		DevStartOps.Container,
	)

	switch len(DevStartOps.LocalSyncDir) {
	case 0:
		associatePath := svcPack.GetAssociatePath()
		if associatePath == "" {
			must(errors.New("'local-sync(-s)' should specify while svc is not associate with local dir"))
		}
		DevStartOps.LocalSyncDir = append(DevStartOps.LocalSyncDir, string(associatePath))

		must(associatePath.Associate(svcPack, common.KubeConfig, true))

		_ = common.NocalhostApp.ReloadSvcCfg(common.WorkloadName, base.SvcType(common.ServiceType), false, false)
	case 1:

		must(dev_dir.DevPath(DevStartOps.LocalSyncDir[0]).Associate(svcPack, common.KubeConfig, true))

		_ = common.NocalhostApp.ReloadSvcCfg(common.WorkloadName, base.SvcType(common.ServiceType), false, false)
	default:
		log.Fatal(errors.New("Can not define multi 'local-sync(-s)'"))
	}
}

// we should clean previous Syncthing
// prevent previous syncthing hold the db lock
func stopPreviousSyncthing() {
	// Clean up previous syncthing
	must(
		common.NocalhostSvc.FindOutSyncthingProcess(
			func(pid int) error {
				return syncthing.Stop(pid, true)
			},
		),
	)
	// kill syncthing process by find find it with terminal
	str := strings.ReplaceAll(common.NocalhostSvc.GetApplicationSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")
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
		common.NocalhostSvc.AppMeta.SvcDevStarting(common.NocalhostSvc.Name, common.NocalhostSvc.Type,
			common.NocalhostApp.Identifier, devModeType),
	)
	must(common.NocalhostSvc.UpdateSvcProfile(func(v2 *profile.SvcProfileV2) error {
		v2.OriginDevContainer = DevStartOps.Container
		return nil
	}))

	// prevent dev status modified but not actually enter dev mode
	var devStartSuccess = false
	var err error
	defer func() {
		if !devStartSuccess {
			log.Infof("Roll backing dev mode...")
			_ = common.NocalhostSvc.AppMeta.SvcDevEnd(common.NocalhostSvc.Name, common.NocalhostSvc.Identifier, common.NocalhostSvc.Type, devModeType)
		}
	}()

	// Only `replace` DevMode needs to disable hpa
	if devModeType.IsReplaceDevMode() {
		log.Info("Disabling hpa...")
		hl, err := common.NocalhostSvc.ListHPA()
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
			if _, err = common.NocalhostSvc.Client.UpdateHPA(&h); err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to update hpa %s", h.Name))
			} else {
				log.Infof("HPA %s has been disabled", h.Name)
			}
		}
	}

	//common.NocalhostSvc.DevModeType = devModeType
	if err = common.NocalhostSvc.BuildPodController().ReplaceImage(context.TODO(), DevStartOps); err != nil {
		log.WarnE(err, "Failed to replace dev container")
		log.Info("Resetting workload...")
		_ = common.NocalhostSvc.DevEnd(true)
		if errors.Is(err, nocalhost.CreatePvcFailed) {
			log.Info("Failed to provision persistent volume due to insufficient resources")
		}
		return err
	}

	if err = common.NocalhostSvc.AppMeta.SvcDevStartComplete(
		common.NocalhostSvc.Name, common.NocalhostSvc.Type, common.NocalhostSvc.Identifier, devModeType,
	); err != nil {
		return err
	}

	utils.Should(common.NocalhostSvc.IncreaseDevModeCount())

	// mark dev start as true
	devStartSuccess = true
	coloredoutput.Success("Dev container has been updated")
	return nil
}

func startPortForwardAfterDevStart(devPodName string) {
	for _, pf := range pfListBeforeDevStart {
		utils.Should(common.NocalhostSvc.PortForward(devPodName, pf.LocalPort, pf.RemotePort, pf.Role))
	}
	must(common.NocalhostSvc.PortForwardAfterDevStart(devPodName, DevStartOps.Container))
}

func must(err error) {
	mustI(err, "")
}

func mustI(err error, info string) {
	if k8serrors.IsForbidden(err) {
		log.FatalE(err, "Permission Denied! Please check that"+
			" your ServiceAccount(KubeConfig) has appropriate permissions.\n\n")
	} else if err != nil {
		log.FatalE(err, info)
	}
}
