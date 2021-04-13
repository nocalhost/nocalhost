/*
Copyright 2021 The Nocalhost Authors.
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
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
)

type convertFlags struct {
	SrcFile     string
	DestFile    string
	FromVersion string
	ToVersion   string
}

var cFlags = convertFlags{}

func init() {
	convertCmd.Flags().
		StringVarP(&cFlags.SrcFile, "srcFile", "f", "", "File needs to get converted")
	convertCmd.Flags().
		StringVarP(&cFlags.DestFile, "destFile", "d", "", "File saves converted config")
	convertCmd.Flags().StringVar(&cFlags.FromVersion, "from-version", "", "From which version")
	convertCmd.Flags().StringVar(&cFlags.ToVersion, "to-version", "", "Convert to which version")
	rootCmd.AddCommand(convertCmd)
}

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert config file between different versions",
	Long:  `Convert config file between different versions`,
	Run: func(cmd *cobra.Command, args []string) {
		if cFlags.SrcFile == "" {
			log.Fatal("-f must be specified")
		}

		if cFlags.FromVersion == "" {
			log.Fatal("--from-version must be specified")
		}

		if cFlags.ToVersion == "" {
			log.Fatal("--to-version must be specified")
		}

		if cFlags.DestFile == "" {
			dir, file := filepath.Split(cFlags.SrcFile)
			cFlags.DestFile = filepath.Join(dir, fmt.Sprintf("v2.%s", file))
		}

		if cFlags.FromVersion == "v1" && cFlags.ToVersion == "v2" {
			err := app.ConvertConfigFileV1ToV2(cFlags.SrcFile, cFlags.DestFile)
			if err != nil {
				log.Fatalf("Failed to convert: %s", err.Error())
			}
			log.Infof("Convert to %s", cFlags.DestFile)
		} else {
			log.Fatal("Unsupported converted")
		}
	},
}
