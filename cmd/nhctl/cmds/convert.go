/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
	convertCmd.Flags().StringVarP(&cFlags.SrcFile, "srcFile", "f", "", "File needs to get converted")
	convertCmd.Flags().StringVarP(&cFlags.DestFile, "destFile", "d", "", "File saves converted config")
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
			must(app.ConvertConfigFileV1ToV2(cFlags.SrcFile, cFlags.DestFile))
			log.Infof("Convert to %s", cFlags.DestFile)
		} else {
			log.Fatal("Unsupported converted")
		}
	},
}
