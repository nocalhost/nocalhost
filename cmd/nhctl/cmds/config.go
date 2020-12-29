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
	"bufio"
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Flags struct {
	Save  bool
	Check bool
}

var flags = Flags{}

func init() {
	configCmd.Flags().BoolVarP(&flags.Save, "save", "s", flags.Save, "save application config file")
	configCmd.Flags().BoolVarP(&flags.Check, "check", "c", flags.Save, "check application config file")
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config [Name]",
	Short: "Application config file",
	Long:  "View, save and check application config file, no flags is view config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		InitApp(applicationName)

		if flags.Save {
			var lines []string
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				var line = scanner.Text()
				lines = append(lines, line)
			}
			file := strings.Join(lines[:], "\n")
			err = nocalhostApp.CheckConfigFile(file)
			if err != nil {
				log.Fatalf("application config file is invalid!\n %s", err.Error())
			}
			//err = nocalhostApp.SaveConfigFile(file)
			//if err != nil {
			//	log.Fatalf("failed to save application config file, \"%s\"", err.Error())
			//}
			fmt.Printf("%s application config file have bean updated!\n", applicationName)
			return
		}

		configFile, err := nocalhostApp.GetConfigFile()
		if err != nil {
			log.Fatalf("failed to get application config file \"%s\"", applicationName)
		}

		if flags.Check {
			err = nocalhostApp.CheckConfigFile(configFile)
			if err != nil {
				log.Fatalf("application config file is invalid!\n %s", err.Error())
			}
			fmt.Printf("%s application config file is valid!\n", applicationName)
			return
		}

		fmt.Println(configFile)
	},
}
