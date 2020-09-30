package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	//install k8s
	//fileSyncCmd.Flags().StringVarP(&sessionName, "session", "s", "", "sync session")
	//fileSyncCmd.Flags().StringVarP(&localFolderName, "local folder", "l", "", "local folder path")
	//fileSyncCmd.Flags().StringVarP(&remoteAddress, "remote address", "r", "", "remote ip address and folder path")
	//fileSyncCmd.Flags().StringVarP(&remoteAddress, "ssh passwd", "p", "", "ssh passwd")
	rootCmd.AddCommand(fileSyncCmd)
}

var fileSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync files to remote Pod in Kubernetes",
	Long: `Sync files to remote Pod in Kubernetes`,
	Run: func(cmd *cobra.Command, args []string) {
		//TO-DO
		fmt.Println("file syncing...")
	},
}