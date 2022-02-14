/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"fmt"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	common2 "nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/utils"
	"time"

	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var installFlags = &app_flags.InstallFlags{}

func init() {

	installCmd.Flags().StringVarP(
		&installFlags.GitUrl, "git-url", "u", "",
		"resources git url",
	)
	installCmd.Flags().StringVarP(
		&installFlags.GitRef, "git-ref", "r", "",
		"resources git ref",
	)
	installCmd.Flags().StringSliceVar(
		&installFlags.ResourcePath, "resource-path", []string{},
		"resources path",
	)
	installCmd.Flags().StringVarP(
		&installFlags.OuterConfig, "outer-config", "c", "",
		"specify a config.yaml in local path",
	)
	installCmd.Flags().StringVar(
		&installFlags.Config, "config", "",
		"specify a config relative to .nocalhost dir",
	)
	installCmd.Flags().StringArrayVarP(
		&installFlags.HelmValueFile, "helm-values", "f", []string{},
		"helm's Value.yaml",
	)
	installCmd.Flags().StringVarP(
		&installFlags.AppType, "type", "t", "", fmt.Sprintf(
			"nocalhost application type: %s, %s, %s, %s, %s, %s or %s",
			appmeta.HelmRepo, appmeta.Helm, appmeta.HelmLocal,
			appmeta.Manifest, appmeta.ManifestGit, appmeta.ManifestLocal, appmeta.KustomizeGit,
		),
	)
	installCmd.Flags().BoolVar(
		&installFlags.HelmWait, "wait", installFlags.HelmWait,
		"wait for completion",
	)
	installCmd.Flags().BoolVar(
		&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall,
		"ignore pre-install",
	)
	installCmd.Flags().StringSliceVar(
		&installFlags.HelmSet, "set", []string{},
		"set values on the command line (can specify multiple "+
			"or separate values with commas: key1=val1,key2=val2)",
	)
	installCmd.Flags().StringVar(
		&installFlags.HelmRepoName, "helm-repo-name", "",
		"chart repository name",
	)
	installCmd.Flags().StringVar(
		&installFlags.HelmRepoUrl, "helm-repo-url", "",
		"chart repository url where to locate the requested chart",
	)
	installCmd.Flags().StringVar(
		&installFlags.HelmRepoVersion, "helm-repo-version", "",
		"chart repository version",
	)
	installCmd.Flags().StringVar(
		&installFlags.HelmChartName, "helm-chart-name", "",
		"chart name",
	)
	installCmd.Flags().StringVar(
		&installFlags.LocalPath, "local-path", "",
		"local path for application",
	)
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install [NAME]",
	Short: "Install k8s application",
	Long:  `Install k8s application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			err             error
			applicationName = args[0]
		)

		must(common2.Prepare())

		if applicationName == _const.DefaultNocalhostApplication {
			log.Error(_const.DefaultNocalhostApplicationOperateErr)
			return
		}

		if installFlags.GitUrl == "" && (installFlags.AppType != string(appmeta.HelmRepo) &&
			installFlags.AppType != string(appmeta.ManifestLocal) &&
			installFlags.AppType != string(appmeta.HelmLocal) &&
			installFlags.AppType != string(appmeta.KustomizeLocal)) {
			log.Fatalf("If app type is not %s , --git-url must be specified", appmeta.HelmRepo)
		}
		if installFlags.AppType == string(appmeta.HelmRepo) {
			if installFlags.HelmChartName == "" {
				log.Fatalf("--helm-chart-name must be specified when using %s", installFlags.AppType)
			}
			if installFlags.HelmRepoUrl == "" && installFlags.HelmRepoName == "" {
				log.Fatalf(
					"--helm-repo-url or "+
						"--helm-repo-name must be specified when using %s", installFlags.AppType,
				)
			}
		}

		log.Info("Installing application...")
		nocalhostApp, err := common.InstallApplication(installFlags, applicationName, common2.KubeConfig, common2.NameSpace)
		must(err)
		log.Infof("Application %s installed", applicationName)

		configV2 := nocalhostApp.GetApplicationConfigV2()

		// Start port forward
		for _, svcProfile := range configV2.ServiceConfigs {
			nhSvc, err := nocalhostApp.InitAndCheckIfSvcExist(svcProfile.Name, svcProfile.Type)
			must(err)
			for _, cc := range svcProfile.ContainerConfigs {
				if cc.Install == nil || len(cc.Install.PortForward) == 0 {
					continue
				}

				svcType := svcProfile.Type
				log.Infof("Starting port-forward for %s %s", svcType, svcProfile.Name)
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)

				var i int
				for i = 0; i < 60; i++ {
					<-time.After(time.Second)
					podName, err = controller.GetDefaultPodName(ctx, nhSvc)
					if err != nil {
						log.WarnE(err, "")
						continue
					}
					log.Infof("Waiting pod %s to be ready", podName)
					pod, err := nocalhostApp.GetClient().GetPod(podName)
					if err != nil {
						log.Info(err.Error())
						continue
					}
					if pod.Status.Phase == "Running" && pod.DeletionTimestamp == nil {
						log.Infof("Pod %s is ready", podName)
						break
					}
				}
				if i == 60 {
					log.Warn("Waiting pod to be ready timeout, continue...")
					continue
				}

				for _, pf := range cc.Install.PortForward {
					lPort, rPort, err := utils.GetPortForwardForString(pf)
					if err != nil {
						log.WarnE(err, "")
						continue
					}
					log.Infof("Port forward %d:%d", lPort, rPort)
					utils.Should(nhSvc.PortForward(podName, lPort, rPort, ""))
				}
			}
		}
	},
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

func mustP(err error) {
	mustPI(err, "")
}

func mustPI(err error, info string) {
	if err != nil {
		log.ErrorE(err, info)
		panic(err)
	}
}
