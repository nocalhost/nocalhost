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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	upgradeCmd.Flags().StringVarP(&installFlags.HelmValueFile, "helm-values", "f", "", "helm's Value.yaml")
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

		initApp(args[0])

		// Check if there are services in developing
		if nocalhostApp.IsAnyServiceInDevMode() {
			log.Fatal("Please make sure all services have exited DevMode")
		}

		// Stop Port-forward
		appProfile, err := nocalhostApp.GetProfile()
		must(err)

		pfListMap := make(map[string][]*profile.DevPortForward, 0)
		for _, svcProfile := range appProfile.SvcProfile {
			nhSvc := initService(svcProfile.ActualName, svcProfile.Type)
			pfList := make([]*profile.DevPortForward, 0)
			for _, pf := range svcProfile.DevPortForwardList {
				if pf.ServiceType == "" {
					pf.ServiceType = svcProfile.Type
				}
				pfList = append(pfList, pf)
				log.Infof("Stopping pf: %d:%d", pf.LocalPort, pf.RemotePort)
				utils.Should(nhSvc.EndDevPortForward(pf.LocalPort, pf.RemotePort))
			}
			if len(pfList) > 0 {
				pfListMap[svcProfile.ActualName] = pfList
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
				nhSvc := initService(svcName, pf.ServiceType)
				podName, err := controller.GetDefaultPodName(ctx, nhSvc.BuildPodController())
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
