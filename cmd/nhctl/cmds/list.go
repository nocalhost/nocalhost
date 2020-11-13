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
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/utils"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list applications",
	Long:  `list applications`,
	Run: func(cmd *cobra.Command, args []string) {
		ListApplications()
	},
}

func ListApplications() {
	n := NocalHost{}
	apps, err := n.GetApplications()
	utils.Mush(err)
	maxLen := 0
	for _, app := range apps {
		if len(app) > maxLen {
			maxLen = len(app)
		}
	}
	fmt.Printf("%-10s\n", "NAME")
	for _, app := range apps {
		fmt.Printf("%-10s\n", app)
	}
}
