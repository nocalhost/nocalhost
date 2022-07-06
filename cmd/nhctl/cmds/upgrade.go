/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func init() {

	upgradeCmd.Flags().StringVarP(&installFlags.GitUrl, "git-url", "u", "", "resources git url")
	upgradeCmd.Flags().StringVarP(&installFlags.GitRef, "git-ref", "r", "", "resources git ref")
	upgradeCmd.Flags().StringSliceVar(&installFlags.ResourcePath, "resource-path", []string{}, "resources path")
	upgradeCmd.Flags().StringVar(&installFlags.Config, "config", "", "specify a config relative to .nocalhost dir")
	upgradeCmd.Flags().StringVarP(&installFlags.OuterConfig, "outer-config", "c", "",
		"specify a config.yaml in local path")
	upgradeCmd.Flags().StringArrayVarP(&installFlags.HelmValueFile, "helm-values", "f", []string{}, "helm's Value.yaml")
	//installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "",
	//fmt.Sprintf("nocalhost application type: %s or %s or %s", app.HelmRepo, app.Helm, app.Manifest))
	upgradeCmd.Flags().StringSliceVar(&installFlags.HelmSet, "set", []string{},
		"set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	upgradeCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "",
		"chart repository url where to locate the requested chart")
	upgradeCmd.Flags().StringVar(&installFlags.HelmRepoVersion, "helm-repo-version", "", "chart repository version")
	upgradeCmd.Flags().StringVar(&installFlags.HelmChartName, "helm-chart-name", "", "chart name")
	upgradeCmd.Flags().StringVar(&installFlags.LocalPath, "local-path", "", "local path for application")
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [NAME]",
	Short: "upgrade k8s application",
	Long:  `upgrade k8s application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		nocalhostApp, err := common.InitApp(args[0])
		must(err)

		// Check if there are services in developing
		if nocalhostApp.IsAnyServiceInDevMode() {
			log.Fatal("Please make sure all services have exited DevMode")
		}

		// Stop Port-forward
		appProfile, err := nocalhostApp.GetProfile()
		must(err)

		pfListMap := make(map[string][]*profile.DevPortForward, 0)
		for _, svcProfile := range appProfile.SvcProfile {
			nhSvc, err := nocalhostApp.InitService(svcProfile.GetName(), svcProfile.GetType())
			must(err)
			pfList := make([]*profile.DevPortForward, 0)
			for _, pf := range svcProfile.DevPortForwardList {
				if pf.ServiceType == "" {
					pf.ServiceType = svcProfile.GetType()
				}
				pfList = append(pfList, pf)
				log.Infof("Stopping pf: %d:%d", pf.LocalPort, pf.RemotePort)
				utils.Should(nhSvc.EndDevPortForward(pf.LocalPort, pf.RemotePort))
			}
			if len(pfList) > 0 {
				pfListMap[svcProfile.GetName()] = pfList
			}
		}

		// todo: Validate flags
		// Prepare for upgrading
		must(nocalhostApp.PrepareForUpgrade(installFlags))

		must(nocalhostApp.Upgrade(installFlags))

		// Restart port forward
		for svcName, pfList := range pfListMap {
			for _, pf := range pfList {
				// find first pod
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
				nhSvc, err := nocalhostApp.InitService(svcName, pf.ServiceType)
				must(err)
				podName, err := controller.GetDefaultPodName(ctx, nhSvc)
				if err != nil {
					log.WarnE(err, "")
					continue
				}
				log.Infof("Starting pf %d:%d for %s", pf.LocalPort, pf.RemotePort, svcName)
				utils.Should(nhSvc.PortForward(podName, pf.LocalPort, pf.RemotePort, pf.Role))
			}
		}
	},
}
