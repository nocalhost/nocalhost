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
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	rootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply [NAME] [MANIFEST]",
	Short: "Apply manifest",
	Long:  `Apply manifest`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.Errorf("%q requires at least 2 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		path := args[1]

		InitApp(applicationName)
		manifests := clientgoutils.LoadValidManifest([]string{path}, []string{})

		err := nocalhostApp.GetClient().ApplyForCreate(manifests, true, app.StandardNocalhostMetas(nocalhostApp.Name, nocalhostApp.GetNamespace()))
		if err != nil {
			log.Fatal(err)
		}
	},
}
