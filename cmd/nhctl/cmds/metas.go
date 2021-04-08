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
	"gopkg.in/yaml.v2"
	"nocalhost/internal/nhctl/app_flags"
	"os"

	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"

	"github.com/spf13/cobra"
)

var metasFlags = &app_flags.ListFlags{}

func init() {
	metasCmd.Flags().BoolVar(&metasFlags.Yaml, "yaml", false, "use yaml as out put, only supports for 'nhctl list'")
	metasCmd.Flags().BoolVar(&metasFlags.Json, "json", false, "use json as out put, only supports for 'nhctl list'")
	rootCmd.AddCommand(metasCmd)
}

var metasCmd = &cobra.Command{
	Use:   "metas [NAME]",
	Short: "List application metas",
	Long:  `List application metas`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 { // list application detail
			applicationName := args[0]
			initApp(applicationName)
			ListApplicationSvc(nocalhostApp)
			os.Exit(0)
		}

		if listFlags.Yaml {
			ListApplicationMetasYaml()
		} else if listFlags.Json {
			ListApplicationMetasJson()
		} else {
			ListApplicationMetas()
		}
	},
}

func ListApplicationMetasJson() {
	appMetas, _ := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	desc := appMetas.Desc()
	marshal, _ := json.Marshal(desc)
	fmt.Print(string(marshal))
}

func ListApplicationMetasYaml() {
	appMetas, _ := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	desc := appMetas.Desc()
	marshal, _ := yaml.Marshal(desc)
	fmt.Print(string(marshal))
}

func ListApplicationMetas() {
	appMetas, err := nocalhost.GetApplicationMetas(nameSpace, kubeConfig)
	if err != nil {
		log.FatalE(err, "Failed to get applications")
	}
	fmt.Printf("%-20s %-20s %-20s\n", "NAME", "STATE", "NAMESPACE")
	for _, meta := range appMetas {
		fmt.Printf("%-20s %-20s %-20s\n", meta.Application, string(meta.ApplicationState), meta.Ns)
	}
}
