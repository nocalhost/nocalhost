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
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
)

var listFlags = &app_flags.ListFlags{}

func init() {
	listCmd.Flags().BoolVar(&listFlags.Yaml, "yaml", false, "use yaml as out put, only supports for 'nhctl list'")
	listCmd.Flags().BoolVar(&listFlags.Json, "json", false, "use json as out put, only supports for 'nhctl list'")
	listCmd.Flags().BoolVar(&listFlags.Full, "full", false, "list application meta full")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list [NAME]",
	Aliases: []string{"ls"},
	Short:   "List applications",
	Long:    `List applications`,
	Run: func(cmd *cobra.Command, args []string) {

		must(Prepare())

		// for earlier version adaption
		if info, _ := nocalhost.GetNsAndApplicationInfo(); info != nil {
			for ns, apps := range info {
				if ns == nameSpace {
					for _, application := range apps {
						_, _ = app.NewApplication(application, nameSpace, kubeConfig, true)
					}
				}
			}
		}

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
		} else if listFlags.Full {
			ListApplicationsFull()
		} else {
			ListApplications()
		}
	},
}

func ListApplicationSvc(napp *app.Application) {
	var data [][]string
	appProfile, _ := napp.GetProfile()
	for _, svcProfile := range appProfile.SvcProfile {
		rols := []string{
			svcProfile.ActualName, strconv.FormatBool(svcProfile.Developing), strconv.FormatBool(svcProfile.Syncing),
			fmt.Sprintf("%v", svcProfile.DevPortForwardList),
			fmt.Sprintf("%s", svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin),
			strconv.Itoa(svcProfile.LocalSyncthingGUIPort),
		}
		data = append(data, rols)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "DEVELOPING", "SYNCING", "DEV-PORT-FORWARDED", "SYNC-PATH", "LOCAL-SYNCTHING-GUI"})

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func ListApplicationsResult() []*Namespace {
	metas, err := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	must(err)

	var result []*Namespace
	ns := &Namespace{
		Namespace:   nameSpace,
		Application: []*ApplicationInfo{},
	}
	for _, meta := range metas {
		ns.Application = append(
			ns.Application, &ApplicationInfo{
				Name: meta.Application,
				Type: meta.ApplicationType,
			},
		)
	}
	result = append(result, ns)
	return result
}

func ListApplicationsFull() {
	metas, err := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	must(err)
	marshal, _ := yaml.Marshal(metas.Desc())
	fmt.Print(string(marshal))
}

func ListApplicationsJson() {
	result := ListApplicationsResult()
	marshal, _ := json.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplicationsYaml() {
	result := ListApplicationsResult()
	marshal, _ := yaml.Marshal(result)
	fmt.Print(string(marshal))
}

func ListApplications() {
	metas, err := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	must(err)
	fmt.Printf("%-20s %-20s %-20s %-20s\n", "NAME", "STATE", "NAMESPACE", "TYPE")
	for _, meta := range metas {
		fmt.Printf("%-20s %-20s %-20s %-20s\n", meta.Application, meta.ApplicationState, meta.Ns, meta.ApplicationType)
	}
}

type Namespace struct {
	Namespace   string
	Application []*ApplicationInfo
}

type ApplicationInfo struct {
	Name string
	Type appmeta.AppType
}
