package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of nhctl",
	Long:  `All software has versions. This is nhctl's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("nhctl version is v0.2.3")
	},
}
