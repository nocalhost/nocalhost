package cmds

import "github.com/spf13/cobra"

var vpnCmd = &cobra.Command{
	Use:   "vpn",
	Short: "vpn",
	Long:  `vpn`,
}

func init() {
	rootCmd.AddCommand(vpnCmd)
}
