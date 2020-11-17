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
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"sync"
	"time"
)

func init() {
	//installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	//installCmd.Flags().StringVarP(&installFlags.Url, "url", "u", "", "resource url")
	//installCmd.Flags().StringVarP(&installFlags.ResourcesDir, "dir", "d", "", "the dir of helm package or manifest")
	//installCmd.Flags().StringVarP(&installFlags.HelmValueFile, "", "f", "", "helm's Value.yaml")
	//installCmd.Flags().StringVarP(&installFlags.AppType, "type", "t", "", "nocalhostApp type: helm or manifest")
	//installCmd.Flags().BoolVar(&installFlags.ForceInstall, "force", installFlags.ForceInstall, "force install")
	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "uninstall application",
	Long:  `uninstall application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		//if nameSpace == "" {
		//	fmt.Println("error: please use -n to specify a kubernetes namespace")
		//	return
		//}
		if !nocalhost.CheckIfApplicationExist(applicationName) {
			fmt.Printf("[error] application \"%s\" not found\n", applicationName)
			os.Exit(1)
		}

		fmt.Println("uninstall application...")
		err := UninstallApplication(applicationName)
		if err != nil {
			printlnErr("failed to uninstall application", err)
			os.Exit(1)
		}
		debug("remove resource files...")
		app, err := nocalhost.GetApplication(applicationName)
		homeDir := app.GetHomeDir()
		err = os.RemoveAll(homeDir)
		if err != nil {
			fmt.Printf("[error] fail to remove nocalhostApp dir %s\n", homeDir)
			os.Exit(1)
		}
		fmt.Printf("application \"%s\" is uninstalled", applicationName)
	},
}

func UninstallApplication(applicationName string) error {
	// delete k8s resources
	app, err := nocalhost.GetApplication(applicationName)
	if err != nil {
		return err
	}

	clientUtil, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)

	if app.AppProfile.DependencyConfigMapName != "" {
		debug("delete config map %s\n", app.AppProfile.DependencyConfigMapName)
		err = clientUtil.DeleteConfigMapByName(app.AppProfile.DependencyConfigMapName, app.AppProfile.Namespace)
		if err != nil {
			return err
		}
	} else {
		debug("no config map found")
	}

	if app.IsHelm() {
		// todo
	} else if app.IsManifest() {
		start := time.Now()
		wg := sync.WaitGroup{}
		resourceDir := app.GetResourceDir()
		files, _, err := GetFilesAndDirs(resourceDir)
		if err != nil {
			return err
		}
		for _, file := range files {
			wg.Add(1)
			fmt.Println("delete " + file)
			go func(fileName string) {
				clientUtil.Delete(fileName, app.GetNamespace())
				wg.Done()
			}(file)

		}
		wg.Wait()
		end := time.Now()
		debug("installing takes %f seconds", end.Sub(start).Seconds())
		return err
	}

	return nil
}
