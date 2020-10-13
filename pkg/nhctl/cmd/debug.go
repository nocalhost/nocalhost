package cmd

import (
	"github.com/spf13/cobra"
)


func init() {
	rootCmd.AddCommand(debugCmd)
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "enter debug model",
	Long: `enter debug model`,
}
