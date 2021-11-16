package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
)

var reconnect pkg.ConnectOptions

func init() {
	reconnectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	reconnectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	reconnectCmd.PersistentFlags().StringArrayVar(&reconnect.Workloads, "workloads", []string{}, "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	reconnectCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	VPNCmd.AddCommand(reconnectCmd)
}

var reconnectCmd = &cobra.Command{
	Use:   "reconnect",
	Short: "reconnect",
	Long:  `reconnect`,
	PreRun: func(*cobra.Command, []string) {
		util.InitLogger(util.Debug)
		if util.IsWindows() {
			driver.InstallWireGuardTunDriver()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !util.IsAdmin() {
			util.RunWithElevated()
			return
		}
		client, err := daemon_client.GetDaemonClient(true)
		if err != nil {
			log.Warn(err)
			return
		}
		must(Prepare())
		err = client.SendVPNOperateCommand(kubeConfig, nameSpace, command.Reconnect, workloads)
		if err != nil {
			log.Warn(err)
		}
	},
}
