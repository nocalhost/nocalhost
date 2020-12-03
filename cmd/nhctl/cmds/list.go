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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list [NAME]",
	Short: "list applications",
	Long:  `list applications`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 { // list application detail
			applicationName := args[0]
			nocalhostApp, err := app.NewApplication(applicationName)
			if err != nil {
				log.Fatal("failed to get application info")
			}
			ListApplicationSvc(nocalhostApp)
			os.Exit(0)
		}
		ListApplications()
	},
}

func ListApplicationSvc(napp *app.Application) {
	var data [][]string
	for _, svcProfile := range napp.AppProfile.SvcProfile {
		rols := []string{svcProfile.Name, strconv.FormatBool(svcProfile.Developing), strconv.FormatBool(svcProfile.Syncing), fmt.Sprintf("%s", svcProfile.DevPortList), fmt.Sprintf("%s", svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin), strconv.Itoa(svcProfile.LocalSyncthingGUIPort)}
		data = append(data, rols)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "DEVELOPING", "SYNCING", "DEV-PORT-FORWARDED", "SYNC-PATH", "LOCAL-SYNCTHING-GUI"})

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func ListApplications() {
	n := nocalhost.NocalHost{}
	apps, err := n.GetApplicationNames()
	utils.Mush(err)
	maxLen := 0
	for _, app := range apps {
		if len(app) > maxLen {
			maxLen = len(app)
		}
	}
	fmt.Printf("%-14s %-14s %-14s\n", "NAME", "INSTALLED", "NAMESPACE")
	for _, app := range apps {
		app2, err := nh.GetApplication(app)
		if err != nil {
			fmt.Printf("%-14s\n", app)
			continue
		}
		profile := app2.AppProfile
		fmt.Printf("%-14s %-14t %-14s\n", app, profile.Installed, profile.Namespace)
	}
}
