package cmds

import (
	"github.com/spf13/cobra"
	"log"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/network"
	"os"
)

func init() {
	connectCmd.Flags().StringVar(&model.Option.Kubeconfig, "kubeconfig", "", "your k8s cluster kubeconfig path")
	connectCmd.Flags().StringVar(&model.Option.ServiceName, "name", "", "service name and deployment name, should be same")
	connectCmd.Flags().StringVar(&model.Option.Namespace, "namespace", "", "service namespace")
	connectCmd.Flags().StringVar(&model.Option.PortPairs, "expose", "", "port pair, remote-port:local-port, such as: service-port1:local-port1,service-port2:local-port2...")
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect",
	Long:  `connect`,
	Run: func(cmd *cobra.Command, args []string) {
		if uid := os.Geteuid(); uid != 0 {
			log.Fatalln("needs sudo privilege, exiting...")
		}
		if err := network.PreCheck(); err != nil {
			panic(err)
		}
		network.Start(model.Option)
	},
}
