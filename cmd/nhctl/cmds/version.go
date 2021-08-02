/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"fmt"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/daemon_common"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	OsArch       = ""
	Version      = ""
	GitCommit    = ""
	BuildTime    = ""
	Branch       = ""
	DevGitCommit = ""
)

func init() {
	daemon_common.Version = Version
	daemon_common.CommitId = GitCommit
	rootCmd.AddCommand(versionCmd)
}

func reformatDate(buildTime string) string {
	t, errTime := time.Parse(time.RFC3339Nano, buildTime)
	if errTime == nil {
		return t.Format(time.ANSIC)
	}
	return buildTime
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of nhctl",
	Long:  `All software has versions. This is nhctl's`,
	Run: func(cmd *cobra.Command, args []string) {
		//Version = "0.3.0"
		if Branch == app.DefaultNocalhostMainBranch {
			fmt.Printf("nhctl: Nocalhost CLI\n")
			fmt.Printf("    Version: %s\n", Version)
			fmt.Printf("    Branch: %s\n", Branch)
			fmt.Printf("    Git commit: %s\n", GitCommit)
			fmt.Printf("    Built time: %s\n", reformatDate(BuildTime))
			fmt.Printf("    Built OS/Arch: %s\n", OsArch)
			fmt.Printf("    Built Go version: %s\n", runtime.Version())
		} else {
			fmt.Printf("nhctl: Nocalhost CLI\n")
			fmt.Printf("    Version: %s\n", Version)
			fmt.Printf("    Branch: %s\n", Branch)
			fmt.Printf("    Git commit: %s\n", DevGitCommit)
			fmt.Printf("    Built time: %s\n", reformatDate(BuildTime))
			fmt.Printf("    Built OS/Arch: %s\n", OsArch)
			fmt.Printf("    Built Go version: %s\n", runtime.Version())
		}
	},
}
