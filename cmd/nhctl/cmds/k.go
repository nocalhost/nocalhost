package cmds

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(kubectlCmd)
}

var kubectlCmd = &cobra.Command{
	Use:   "k",
	Short: "kubectl",
	Long:  `kubectl`,
}
