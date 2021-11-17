package cmds

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
)

var workloads string

func init() {
	connectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	connectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	connectCmd.Flags().StringVarP(&workloads, "workloads", "", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	vpnCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect",
	Long:  `connect`,
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
		err = client.SendVPNOperateCommand(kubeConfig, nameSpace, command.Connect, workloads)
		if err != nil {
			log.Warn(err)
		}
	},
}
