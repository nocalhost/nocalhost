package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var localPort string

func init() {
	portForwardCmd.Flags().StringVarP(&localPort, "local", "p", "10000", "local port to forward")
	rootCmd.AddCommand(portForwardCmd)
}

var portForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Forward local port to remote pod'port",
	Long: `Forward local port to remote pod'port`,
	Run: func(cmd *cobra.Command, args []string) {
		//TO-DO
		fmt.Printf("port forwarding...local port : %s\n", localPort)
	},
}
