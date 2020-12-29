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
	"nocalhost/internal/nhctl/nocalhost"
	"os"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type InstallFlags struct {
	*EnvSettings
	GitUrl  string // resource url
	GitRef  string
	AppType string
	//ResourcesDir  string
	HelmValueFile    string
	ForceInstall     bool
	IgnorePreInstall bool
	HelmSet          []string
	HelmRepoName     string
	HelmRepoUrl      string
	HelmRepoVersion  string
	HelmChartName    string
	HelmWait         bool
	OuterConfig      string
	Config           string
	ResourcePath     []string
}

var installFlags = InstallFlags{
	EnvSettings: settings,
}

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&installFlags.GitUrl, "git-url", "u", "", "resources git url")
	installCmd.Flags().StringVarP(&installFlags.GitRef, "git-ref", "r", "", "resources git ref")
	installCmd.Flags().StringSliceVar(&installFlags.ResourcePath, "resource-path", []string{}, "resources path")
	installCmd.Flags().StringVarP(&installFlags.OuterConfig, "outer-config", "c", "", "specify a config.yaml")
	installCmd.Flags().StringVar(&installFlags.Config, "config", "", "specify a config relative to .nocalhost dir")
	installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "helm-values", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", "nocalhostApp type: helm or helm-repo or manifest")
	installCmd.Flags().BoolVar(&installFlags.HelmWait, "wait", installFlags.HelmWait, "wait for completion")
	installCmd.Flags().BoolVar(&installFlags.IgnorePreInstall, "ignore-pre-install", installFlags.IgnorePreInstall, "ignore pre-install")
	installCmd.Flags().StringSliceVar(&installFlags.HelmSet, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	installCmd.Flags().StringVar(&installFlags.HelmRepoName, "helm-repo-name", "", "chart repository name")
	installCmd.Flags().StringVar(&installFlags.HelmRepoUrl, "helm-repo-url", "", "chart repository url where to locate the requested chart")
	installCmd.Flags().StringVar(&installFlags.HelmRepoVersion, "helm-repo-version", "", "chart repository version")
	installCmd.Flags().StringVar(&installFlags.HelmChartName, "helm-chart-name", "", "chart name")
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
		//if settings.Debug {
		//	log.SetLevel(logrus.DebugLevel)
		//}
		applicationName := args[0]
		if installFlags.GitUrl == "" && installFlags.AppType != string(app.HelmRepo) {
			log.Fatalf("if app type is not %s , --git-url must be specified", app.HelmRepo)
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
			log.Fatalf("application \"%s\" already exists", applicationName)
		}

		log.Info("installing application...")
		err = InstallApplication(applicationName)
		if err != nil {
			fmt.Printf("failed to install application : %s\n", err.Error())
			log.Debug("failed to install application, clean up resources...")
			err = nocalhost.CleanupAppFiles(applicationName)
			if err != nil {
				fmt.Errorf("failed to clean up:%v", err)
			} else {
				log.Debug("resources have been clean up")
			}
			os.Exit(-1)
		} else {
			fmt.Printf("application \"%s\" installed\n", applicationName)
		}
	},
}

func InstallApplication(applicationName string) error {
	var err error

	nocalhostApp, err = app.BuildApplication(applicationName)
	if err != nil {
		return err
	}

	err = nocalhostApp.InitClient(settings.KubeConfig, nameSpace)
	if err != nil {
		return err
	}

	if installFlags.GitUrl != "" {
		err = nocalhostApp.DownloadResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
		if err != nil {
			log.Debugf("failed to clone : %s, ref: %s\n", installFlags.GitUrl, installFlags.GitRef)
			return err
		}
	}

	err = nocalhostApp.InitConfig(installFlags.OuterConfig, installFlags.Config)
	if err != nil {
		return err
	}

	// flags which no config mush specify
	if installFlags.AppType != "" {
		nocalhostApp.AppProfile.AppType = app.AppType(installFlags.AppType)
	}
	if len(installFlags.ResourcePath) != 0 {
		nocalhostApp.AppProfile.ResourcePath = installFlags.ResourcePath
	}
	nocalhostApp.AppProfile.Save()

	appType, err := nocalhostApp.GetType()
	if appType == "" {
		return errors.New("--type must be specified")
	}

	flags := &app.HelmFlags{
		//Debug:  installFlags.Debug,
		Values:   installFlags.HelmValueFile,
		Set:      installFlags.HelmSet,
		Wait:     installFlags.HelmWait,
		Chart:    installFlags.HelmChartName,
		RepoUrl:  installFlags.HelmRepoUrl,
		RepoName: installFlags.HelmRepoName,
		Version:  installFlags.HelmRepoVersion,
	}
	err = nocalhostApp.InstallDepConfigMap(appType)
	if err != nil {
		return errors.Wrap(err, "failed to install dep config map")
	}
	switch appType {
	case app.Helm:
		err = nocalhostApp.InstallHelmInGit(applicationName, flags)
	case app.HelmRepo:
		err = nocalhostApp.InstallHelmInRepo(applicationName, flags)
	case app.Manifest:
		err = nocalhostApp.InstallManifest()
	default:
		return errors.New(fmt.Sprintf("unsupported application type, must be %s, %s or %s", app.Helm, app.HelmRepo, app.Manifest))
	}
	if err != nil {
		return err
	}

	nocalhostApp.SetAppType(appType)
	err = nocalhostApp.SetInstalledStatus(true)
	if err != nil {
		return errors.New("failed to update \"installed\" status")
	}
	return nil
}
