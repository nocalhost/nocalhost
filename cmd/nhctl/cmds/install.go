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
	"io/ioutil"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"time"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var installFlags = &app_flags.InstallFlags{}

func init() {

	installCmd.Flags().StringVarP(
		&nameSpace, "namespace", "n", "",
		"kubernetes namespace",
	)
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
	installCmd.Flags().StringVarP(
		&installFlags.HelmValueFile, "helm-values", "f", "",
		"helm's Value.yaml",
	)
	installCmd.Flags().StringVarP(
		&installFlags.AppType, "type", "t", "", fmt.Sprintf(
			"nocalhost application type: %s, %s, %s, %s, %s or %s",
			appmeta.HelmRepo, appmeta.Helm, appmeta.HelmLocal,
			appmeta.Manifest, appmeta.ManifestLocal, appmeta.KustomizeGit,
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

		must(Prepare())

		if applicationName == nocalhost.DefaultNocalhostApplication {
			log.Error(nocalhost.DefaultNocalhostApplicationOperateErr)
			return
		}

		if installFlags.GitUrl == "" && (installFlags.AppType != string(appmeta.HelmRepo) &&
			installFlags.AppType != string(appmeta.ManifestLocal) &&
			installFlags.AppType != string(appmeta.HelmLocal)) {
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
		must(InstallApplication(applicationName))
		log.Infof("Application %s installed", applicationName)

		profileV2, err := nocalhostApp.GetProfile()
		must(err)

		// Start port forward
		for _, svcProfile := range profileV2.SvcProfile {
			nhSvc := initService(svcProfile.ActualName, svcProfile.Type)
			for _, cc := range svcProfile.ContainerConfigs {
				if cc.Install == nil || len(cc.Install.PortForward) == 0 {
					continue
				}

				svcType := svcProfile.Type
				log.Infof("Starting port-forward for %s %s", svcType, svcProfile.ActualName)
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
				podController := nhSvc.BuildPodController()
				if podController == nil {
					log.WarnE(errors.New("Pod controller is nil"), "")
					continue
				}
				podName, err := controller.GetDefaultPodName(ctx, nhSvc.BuildPodController())
				if err != nil {
					log.WarnE(err, "")
					continue
				}
				for _, pf := range cc.Install.PortForward {
					lPort, rPort, err := controller.GetPortForwardForString(pf)
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

func InstallApplication(applicationName string) error {
	var err error

	log.Logf("KubeConfig path: %s", kubeConfig)
	bys, err := ioutil.ReadFile(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "")
	}
	log.Logf("KubeConfig content: %s", string(bys))

	// build Application will create the application meta and it's secret
	// init the application's config
	if nocalhostApp, err = app.BuildApplication(applicationName, installFlags, kubeConfig, nameSpace); err != nil {
		return err
	}

	// if init appMeta successful, then should remove all things while fail
	defer func() {
		if err != nil {
			coloredoutput.Fail(err.Error())
			log.LogE(err)
			utils.Should(nocalhostApp.Uninstall())
		}
	}()

	appType := nocalhostApp.GetType()
	if appType == "" {
		return errors.New("--type must be specified")
	}

	// add helmValue in config
	helmValue := nocalhostApp.GetApplicationConfigV2().HelmValues
	for _, v := range helmValue {
		installFlags.HelmSet = append([]string{fmt.Sprintf("%s=%s", v.Key, v.Value)}, installFlags.HelmSet...)
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

	err = nocalhostApp.Install(flags)
	_ = nocalhostApp.CleanUpTmpResources()
	return err
}

func must(err error) {
	mustI(err, "")
}

func mustI(err error, info string) {
	if err != nil {
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
