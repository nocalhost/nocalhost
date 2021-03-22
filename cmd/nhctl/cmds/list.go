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
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/app_flags"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var listFlags = &app_flags.ListFlags{}

func init() {
	listCmd.Flags().BoolVar(&listFlags.Yaml, "yaml", false, "use yaml as out put, only supports for 'nhctl list'")
	listCmd.Flags().BoolVar(&listFlags.Json, "json", false, "use json as out put, only supports for 'nhctl list'")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list [NAME]",
	Aliases: []string{"ls"},
	Short:   "List applications",
	Long:    `List applications`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 { // list application detail
			applicationName := args[0]
			initApp(applicationName)
			ListApplicationSvc(nocalhostApp)
			os.Exit(0)
		}

		if listFlags.Yaml {
			ListApplicationsYaml()
		} else if listFlags.Json {
			ListApplicationsJson()
		} else {
			ListApplications()
		}
	},
}

func ListApplicationSvc(napp *app.Application) {
	var data [][]string
	for _, svcProfile := range napp.AppProfileV2.SvcProfile {
		rols := []string{svcProfile.ActualName, strconv.FormatBool(svcProfile.Developing), strconv.FormatBool(svcProfile.Syncing), fmt.Sprintf("%v", svcProfile.DevPortForwardList), fmt.Sprintf("%s", svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin), strconv.Itoa(svcProfile.LocalSyncthingGUIPort)}
		data = append(data, rols)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "DEVELOPING", "SYNCING", "DEV-PORT-FORWARDED", "SYNC-PATH", "LOCAL-SYNCTHING-GUI"})

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func ListApplicationsReuslt() []*Namespace {
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		log.FatalE(err, "Failed to get applications")
	}

	result := []*Namespace{}
	for ns, appList := range appMap {
		for _, appName := range appList {
			app2, err := app.NewApplication(appName, ns, "", false)
			if err != nil {
				continue
			}

			profile := app2.AppProfileV2

			if !profile.Installed {
				continue
			}
			var namespace *Namespace = nil
			var index = 0
			for rni, rns := range result {
				if rns.Namespace == profile.Namespace {
					namespace = rns
					index = rni
					break
				}
			}
			if namespace == nil {
				namespace = &Namespace{
					Namespace:   profile.Namespace,
					Application: []*ApplicationInfo{},
				}
				apps := append(namespace.Application, &ApplicationInfo{
					Name: appName,
					Type: profile.AppType,
				})
				namespace.Application = apps
				result = append(result, namespace)
			} else {
				apps := append(namespace.Application, &ApplicationInfo{
					Name: appName,
					Type: profile.AppType,
				})
				namespace.Application = apps

				result[index] = namespace
			}
		}
	}
	return result
}

func ListApplicationsJson() {
	result := ListApplicationsReuslt()
	marshal, _ := json.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplicationsYaml() {
	result := ListApplicationsReuslt()
	marshal, _ := yaml.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplications() {
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		log.FatalE(err, "Failed to get applications")
	}
	fmt.Printf("%-14s %-14s %-14s %-14s\n", "NAME", "INSTALLED", "NAMESPACE", "TYPE")
	for ns, appList := range appMap {
		for _, appName := range appList {
			app2, err := app.NewApplication(appName, ns, "", false)
			if err != nil {
				log.WarnE(err, "Failed to new application")
				fmt.Printf("%-14s\n", appName)
				continue
			}
			profile := app2.AppProfileV2
			fmt.Printf("%-14s %-14t %-14s %-14s\n", appName, profile.Installed, profile.Namespace, profile.AppType)
		}
	}
}

type Namespace struct {
	Namespace   string
	Application []*ApplicationInfo
}

type ApplicationInfo struct {
	Name string
	Type string
}
