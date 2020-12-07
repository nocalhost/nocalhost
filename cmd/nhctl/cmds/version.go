package cmds

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	OsArch    = ""
	Version   = ""
	GitCommit = ""
	BuildTime = ""
)

func init() {
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
		fmt.Printf("nhctl: Nocalhost CLI\n")
		fmt.Printf("    Version: %s\n", Version)
		fmt.Printf("    Git commit: %s\n", GitCommit)
		fmt.Printf("    Built: %s\n", reformatDate(BuildTime))
		fmt.Printf("    OS/Arch: %s\n", OsArch)
		fmt.Printf("    Go version: %s\n", runtime.Version())
	},
}
