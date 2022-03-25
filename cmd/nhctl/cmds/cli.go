/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/ui"
)

func init() {
	rootCmd.AddCommand(cli2Command)
}

var cli2Command = &cobra.Command{
	Use:   "cli",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ui.RunTviewApplication()
	},
}
