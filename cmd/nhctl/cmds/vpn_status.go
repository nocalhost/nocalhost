package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
	"sigs.k8s.io/yaml"
)

func init() {
	vpnStatusCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	vpnStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	vpnStatusCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	vpnStatusCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(vpnStatusCmd)
}

var vpnStatusCmd = &cobra.Command{
	Use:   "reset",
	Short: "reset",
	Long:  `reset`,
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
		result, err := client.SendVPNStatusCommand(kubeConfig, nameSpace, workloads)
		if err != nil {
			log.Warn(err)
			return
		}
		marshal, _ := yaml.Marshal(result)
		println(string(marshal))
	},
}
