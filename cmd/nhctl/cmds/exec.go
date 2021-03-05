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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"nocalhost/pkg/nhctl/log"
)

type ExecFlags struct {
	CommonFlags
	Commands []string
}

var execFlags = ExecFlags{}

func init() {
	execCmd.Flags().StringArrayVarP(&execFlags.Commands, "command", "c", nil, "command to execute in container")
	execCmd.Flags().StringVarP(&execFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	rootCmd.AddCommand(execCmd)
}

var execCmd = &cobra.Command{
	Use:   "exec [NAME]",
	Short: "Execute a command in container",
	Long:  `Execute a command in container`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		execFlags.AppName = args[0]
		InitAppAndCheckIfSvcExist(execFlags.AppName, execFlags.SvcName)

		err = nocalhostApp.Exec(execFlags.SvcName, "", execFlags.Commands)
		if err != nil {
			log.Fatalf("Failed to exec command : %s", err.Error())
		}
	},
}
