/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/dev"
)

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.AddCommand(dev.DevStartCmd)
	debugCmd.AddCommand(dev.DevEndCmd)
}

var debugCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start DevMode",
	Long:  `Start DevMode`,
}
