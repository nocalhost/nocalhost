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
)

type PluginArg struct {
	Deployment string
}

var pluginArg PluginArg

func init() {
	PluginCmd.Flags().StringVarP(&pluginArg.Deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	rootCmd.AddCommand(PluginCmd)
}

var PluginCmd = &cobra.Command{
	Use:   "plugin get [NAME]",
	Short: "Plugin get application status",
	Long:  `Plugin get application status`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[1]
		InitApp(applicationName)
		fmt.Println(nocalhostApp.GetPluginDescription(pluginArg.Deployment))
	},
}
