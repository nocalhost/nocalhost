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
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

type ConfigEditFlags struct {
	CommonFlags
	Content string
}

var configEditFlags = ConfigEditFlags{}

func init() {
	configEditCmd.Flags().StringVarP(&configEditFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	configEditCmd.Flags().StringVarP(&configEditFlags.Content, "content", "c", "", "base64 encode json content")
	configCmd.AddCommand(configEditCmd)
}

var configEditCmd = &cobra.Command{
	Use:   "edit [Name]",
	Short: "edit service config",
	Long:  "edit service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		configEditFlags.AppName = args[0]
		InitAppAndCheckIfSvcExist(configEditFlags.AppName, configEditFlags.SvcName)

		if len(configEditFlags.Content) == 0 {
			log.Fatal("--content required")
		}

		bys, err := base64.StdEncoding.DecodeString(configEditFlags.Content)
		if err != nil {
			log.Fatalf("--content must be a valid base64 string: %s", err.Error())
		}

		svcConfig := &app.ServiceDevOptions{}
		err = json.Unmarshal(bys, svcConfig)
		if err != nil {
			log.Fatalf("fail to unmarshal content: %s", err.Error())
		}
		err = nocalhostApp.SaveSvcConfig(configEditFlags.SvcName, svcConfig)
		if err != nil {
			log.FatalE(err, "fail to save svc config")
		}
	},
}
