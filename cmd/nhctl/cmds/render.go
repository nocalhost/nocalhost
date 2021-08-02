/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
)

var renderOps = &RenderOps{}

type RenderOps struct {
	envPath string
}

func init() {
	renderCmd.Flags().StringVarP(&renderOps.envPath, "env path", "e", "", "the env file for render injection")
	rootCmd.AddCommand(renderCmd)
}

var renderCmd = &cobra.Command{
	Use:   "render [NAME]",
	Short: "Render the file for debugging",
	Long:  `Render the file for debugging`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		render, err := envsubst.Render(fp.NewFilePath(args[0]), fp.NewFilePath(renderOps.envPath))
		must(errors.Wrap(err, ""))
		fmt.Print(render)
	},
}
