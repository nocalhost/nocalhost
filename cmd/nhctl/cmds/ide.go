/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(ideCmd)
}

var ideCmd = &cobra.Command{
	Use:   "ide",
	Short: "used by IDE plugin, such as vscode or jetbrains",
	Long:  `used by IDE plugin, such as vscode or jetbrains`,
}
