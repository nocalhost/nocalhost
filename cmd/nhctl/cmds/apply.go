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

		initApp(applicationName)
		manifests := clientgoutils.LoadValidManifest([]string{path}, []string{})

		err := nocalhostApp.GetClient().Apply(
			manifests, true,
			app.StandardNocalhostMetas(nocalhostApp.Name, nocalhostApp.NameSpace), "",
		)
		if err != nil {
			log.Fatal(err)
		}
	},
}
