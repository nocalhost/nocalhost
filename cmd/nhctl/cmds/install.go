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
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/log"
	"os"
)

type InstallFlags struct {
	*EnvSettings
	GitUrl  string // resource url
	AppType string
	//ResourcesDir  string
	HelmValueFile    string
	ForceInstall     bool
	IgnorePreInstall bool
	HelmSet          string
	HelmRepoName     string
	HelmRepoUrl      string
	HelmChartName    string
	HelmWait         bool
	Config           string
	ResourcePath     string
}

var installFlags = InstallFlags{
	EnvSettings: settings,
}

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&installFlags.GitUrl, "git-url", "u", "", "resources git url")
	installCmd.Flags().StringVar(&installFlags.ResourcePath, "resource-path", "", "resources path")
	installCmd.Flags().StringVarP(&installFlags.Config, "config", "c", "", "specify a config.yaml")
	//installCmd.Flags().StringVarP(&installFlags.ResourcesDir, "dir", "d", "", "the dir of helm package or manifest")
	installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "helm-values", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", "nocalhostApp type: helm or helm-repo or manifest")
	//installCmd.Flags().BoolVar(&installFlags.ForceInstall, "force", installFlags.ForceInstall, "force install")
	installCmd.Flags().BoolVar(&installFlags.HelmWait, "wait", installFlags.HelmWait, "wait for completion")
	installCmd.Flags().BoolVar(&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall, "ignore pre-install")
	installCmd.Flags().StringVar(&installFlags.HelmSet, "set", "", "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	installCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	installCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "", "chart repository url where to locate the requested chart")
	installCmd.Flags().StringVar(&installFlags.HelmChartName, "helm-chart-name", "", "chart name")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install [NAME]",
	Short: "install k8s application",
	Long:  `install k8s application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		var err error
		if installFlags.GitUrl == "" && installFlags.AppType != string(app.HelmRepo) {
			fmt.Println("error: if app type is not helm-repo , --url must be specified")
			os.Exit(1)
		}
		if installFlags.AppType == string(app.HelmRepo) {
			if installFlags.HelmChartName == "" {
				fmt.Println("error: --helm-chart-name must be specified")
				os.Exit(1)
			}
			if installFlags.HelmRepoUrl == "" && installFlags.HelmRepoName == "" {
				fmt.Println("error: --helm-repo-url or --helm-repo-name must be specified")
				os.Exit(1)
			}
		}
		if nh.CheckIfApplicationExist(applicationName) {
			fmt.Printf("[error] application \"%s\" already exists\n", applicationName)
			os.Exit(1)
		}

		fmt.Println("install application...")
		err = InstallApplication(applicationName)
		if err != nil {
			printlnErr("failed to install application", err)
			log.Debug("clean up resources...")
			err = nh.CleanupAppFiles(applicationName)
			if err != nil {
				fmt.Printf("[error] failed to clean up:%v\n", err)
			} else {
				log.Debug("resources have been clean up")
			}
			os.Exit(1)
		} else {
			fmt.Printf("application \"%s\" is installed", applicationName)
		}
	},
}

func InstallApplication(applicationName string) error {

	var (
		err error
	)

	nocalhostApp, err = app.BuildApplication(applicationName)
	if err != nil {
		return err
	}

	err = nocalhostApp.InitClient(settings.KubeConfig, nameSpace)
	if err != nil {
		return err
	}

	// init application dir
	if installFlags.GitUrl != "" {
		err = nocalhostApp.DownloadResourcesFromGit(installFlags.GitUrl)
		if err != nil {
			fmt.Printf("[error] failed to clone : %s\n", installFlags.GitUrl)
			return err
		}
	}

	err = nocalhostApp.InitConfig(installFlags.Config)
	if err != nil {
		return err
	}

	// flags which no config mush specify
	if installFlags.AppType != "" {
		nocalhostApp.AppProfile.AppType = app.AppType(installFlags.AppType)
	}
	if installFlags.ResourcePath != "" {
		nocalhostApp.AppProfile.ResourcePath = installFlags.ResourcePath
	}
	nocalhostApp.AppProfile.Save()

	appType, err := nocalhostApp.GetType()
	if appType == "" {
		return errors.New("--type mush be specified")
	}

	log.Debugf("[nh config] nocalhostApp type: %s", appType)
	flags := &app.HelmFlags{
		//Debug:  installFlags.Debug,
		Values:   installFlags.HelmValueFile,
		Set:      installFlags.HelmSet,
		Wait:     installFlags.HelmWait,
		Chart:    installFlags.HelmChartName,
		RepoUrl:  installFlags.HelmRepoUrl,
		RepoName: installFlags.HelmRepoName,
	}
	switch appType {
	case app.Helm:
		dir := nocalhostApp.GetResourceDir()
		if dir == "" {
			return errors.New("--resource-path mush be specified")
		}
		err = nocalhostApp.InstallHelm(applicationName, flags)
	case app.HelmRepo:
		err = nocalhostApp.InstallHelmRepo(applicationName, flags)
	case app.Manifest:
		dir := nocalhostApp.GetResourceDir()
		if dir == "" {
			return errors.New("--resource-path mush be specified")
		}
		err = nocalhostApp.InstallManifest()
	default:
		return errors.New("unsupported application type, it mush be helm or helm-repo or manifest")
	}
	if err != nil {
		return err
	}

	nocalhostApp.SetAppType(appType)
	err = nocalhostApp.SetInstalledStatus(true)
	if err != nil {
		return errors.New("fail to update \"installed\" status")
	}
	return nil
}
