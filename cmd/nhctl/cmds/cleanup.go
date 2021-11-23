/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost_cleanup"
)

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "upgrade k8s application",
	Long:  `upgrade k8s application`,
	Run: func(cmd *cobra.Command, args []string) {
		must(nocalhost_cleanup.CleanUp(true))
	},
}
