package cmds

import (
	"fmt"

	"github.com/spf13/cobra"
)

var GIT_COMMIT_SHA string
var GIT_TAG string

type VersionInfo struct {
	Version    string `json:"version" yaml:"version"`
	GitVersion string `json:"gitVersion"`
	GitCommit  string `json:"gitCommit"`
	BuildDate  string `json:"buildDate"`
	GoVersion  string `json:"goVersion"`
	Compiler   string `json:"compiler"`
	Platform   string `json:"platform"`
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of nhctl",
	Long:  `All software has versions. This is nhctl's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nhctl version is: %s\n", GIT_TAG)
		fmt.Printf("build commit sha: %s\n", GIT_COMMIT_SHA)
	},
}
