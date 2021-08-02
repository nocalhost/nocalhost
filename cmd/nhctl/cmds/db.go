/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"github.com/spf13/cobra"
)

var (
	appName          string
	levelDbKey       string
	levelDbValue     string
	levelDbValueFile string
)

func init() {
	rootCmd.AddCommand(dbCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Get leveldb data",
	Long:  `Get leveldb data`,
}
