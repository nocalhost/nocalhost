/*
Copyright 2021 The Nocalhost Authors.
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
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func init() {

	//upgradeCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	upgradeCmd.Flags().StringVarP(&installFlags.GitUrl, "git-url", "u", "", "resources git url")
	upgradeCmd.Flags().StringVarP(&installFlags.GitRef, "git-ref", "r", "", "resources git ref")
	upgradeCmd.Flags().StringSliceVar(&installFlags.ResourcePath, "resource-path", []string{}, "resources path")
	//installCmd.Flags().StringVarP(&installFlags.OuterConfig, "outer-config", "c", "", "specify a config.yaml in local path")
	upgradeCmd.Flags().StringVar(&installFlags.Config, "config", "", "specify a config relative to .nocalhost dir")
	//installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "helm-values", "f", "", "helm's Value.yaml")
	//installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", fmt.Sprintf("nocalhost application type: %s or %s or %s", app.HelmRepo, app.Helm, app.Manifest))
	//installCmd.Flags().BoolVar(&installFlags.HelmWait, "wait", installFlags.HelmWait, "wait for completion")
	//installCmd.Flags().BoolVar(&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall, "ignore pre-install")
	//installCmd.Flags().StringSliceVar(&installFlags.HelmSet, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	upgradeCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "", "chart repository url where to locate the requested chart")
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
		if err != nil {
			log.FatalE(err, "")
		}
		pfListMap := make(map[string][]*profile.DevPortForward, 0)
		for _, svcProfile := range appProfile.SvcProfile {
			pfList := make([]*profile.DevPortForward, 0)
			for _, pf := range svcProfile.DevPortForwardList {
				pfList = append(pfList, pf)
				log.Infof("Stopping pf: %d:%d", pf.LocalPort, pf.RemotePort)
				err = nocalhostApp.EndDevPortForward(svcProfile.ActualName, pf.LocalPort, pf.RemotePort)
				if err != nil {
					log.WarnE(err, "")
				}
			}
			if len(pfList) > 0 {
				pfListMap[svcProfile.ActualName] = pfList
			}
		}

		// todo: Validate flags
		err = nocalhostApp.Upgrade(installFlags)
		if err != nil {
			log.FatalE(err, fmt.Sprintf("Failed to upgrade application"))
		}

		// Restart port forward
		for svcName, pfList := range pfListMap {
			// find first pod
			podList, err := nocalhostApp.GetPodsFromDeployment(svcName)
			if err != nil {
				log.WarnE(err, "")
				continue
			}
			if podList == nil || len(podList.Items) == 0 {
				log.Warnf("No pod found in %s", svcName)
				continue
			}
			podName := podList.Items[0].Name
			for _, pf := range pfList {
				log.Infof("Starting pf %d:%d for %s", pf.LocalPort, pf.RemotePort, svcName)
				if err = nocalhostApp.PortForward(svcName, podName, pf.LocalPort, pf.RemotePort); err != nil {
					log.WarnE(err, "")
				}
			}
		}
	},
}
