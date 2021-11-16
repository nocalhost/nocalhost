package cmds

import "github.com/spf13/cobra"

var VPNCmd = &cobra.Command{
	Use:   "vpn",
	Short: "vpn",
	Long:  `vpn`,
}

func init() {
	rootCmd.AddCommand(VPNCmd)
}
