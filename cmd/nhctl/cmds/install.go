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
	"io/ioutil"
	"os"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var installFlags = &app_flags.InstallFlags{}

func init() {

	installCmd.Flags().StringVarP(&installFlags.Namespace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&installFlags.GitUrl, "git-url", "u", "", "resources git url")
	installCmd.Flags().StringVarP(&installFlags.GitRef, "git-ref", "r", "", "resources git ref")
	installCmd.Flags().StringSliceVar(&installFlags.ResourcePath, "resource-path", []string{}, "resources path")
	installCmd.Flags().StringVarP(&installFlags.OuterConfig, "outer-config", "c", "", "specify a config.yaml in local path")
	installCmd.Flags().StringVar(&installFlags.Config, "config", "", "specify a config relative to .nocalhost dir")
	installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "helm-values", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", fmt.Sprintf("nocalhost application type: %s or %s or %s", app.HelmRepo, app.Helm, app.Manifest))
	installCmd.Flags().BoolVar(&installFlags.HelmWait, "wait", installFlags.HelmWait, "wait for completion")
	installCmd.Flags().BoolVar(&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall, "ignore pre-install")
	installCmd.Flags().StringSliceVar(&installFlags.HelmSet, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	installCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	installCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "", "chart repository url where to locate the requested chart")
	installCmd.Flags().StringVar(&installFlags.HelmRepoVersion, "helm-repo-version", "", "chart repository version")
	installCmd.Flags().StringVar(&installFlags.HelmChartName, "helm-chart-name", "", "chart name")
	installCmd.Flags().StringVar(&installFlags.LocalPath, "local-path", "", "local path for application")
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
		var err error
		applicationName := args[0]
		if installFlags.GitUrl == "" && installFlags.AppType != string(app.HelmRepo) {
			log.Fatalf("If app type is not %s , --git-url must be specified", app.HelmRepo)
		}
		if installFlags.AppType == string(app.HelmRepo) {
			if installFlags.HelmChartName == "" {
				log.Fatalf("--helm-chart-name must be specified when using %s", installFlags.AppType)
			}
			if installFlags.HelmRepoUrl == "" && installFlags.HelmRepoName == "" {
				log.Fatalf("--helm-repo-url or --helm-repo-name must be specified when using %s", installFlags.AppType)
			}
		}
		if nocalhost.CheckIfApplicationExist(applicationName) {
			log.Fatalf("Application %s already exists", applicationName)
		}

		log.Info("Installing application...")
		err = InstallApplication(applicationName)
		if err != nil {
			log.Warnf("Failed to install application : %s", err.Error())
			log.Debug("Cleaning up resources...")
			err = nocalhost.CleanupAppFiles(applicationName)
			if err != nil {
				log.Errorf("Failed to clean up: %v", err)
			} else {
				log.Debug("Resources have been clean up")
			}
			os.Exit(-1)
		} else {
			log.Infof("Application %s installed", applicationName)
		}
	},
}

func InstallApplication(applicationName string) error {
	var err error

	installFlags.EnvSettings = settings

	log.Logf("KubeConfig path: %s", installFlags.KubeConfig)
	bys, err := ioutil.ReadFile(installFlags.KubeConfig)
	if err != nil {
		return errors.Wrap(err, "")
	}
	log.Logf("KubeConfig content: %s", string(bys))

	nocalhostApp, err = app.BuildApplication(applicationName, installFlags)
	if err != nil {
		return err
	}

	appType := nocalhostApp.GetType()
	if appType == "" {
		return errors.New("--type must be specified")
	}

	flags := &app.HelmFlags{
		Values:   installFlags.HelmValueFile,
		Set:      installFlags.HelmSet,
		Wait:     installFlags.HelmWait,
		Chart:    installFlags.HelmChartName,
		RepoUrl:  installFlags.HelmRepoUrl,
		RepoName: installFlags.HelmRepoName,
		Version:  installFlags.HelmRepoVersion,
	}

	err = nocalhostApp.Install(context.TODO(), flags)
	if err != nil {
		return err
	}

	return nil
}
