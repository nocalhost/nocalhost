package cmds

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
)

//var reconnect pkg.ConnectOptions

func init() {
	reconnectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	reconnectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	reconnectCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	reconnectCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(reconnectCmd)
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
		// if not sudo and sudo daemon is not running, needs sudo permission
		if !util.IsAdmin() && !util.IsSudoDaemonServing() {
			util.RunWithElevated()
			return
		}
		_, err := daemon_client.GetDaemonClient(true)
		if err != nil {
			log.Warn(err)
			return
		}
		client, err := daemon_client.GetDaemonClient(false)
		if err != nil {
			log.Warn(err)
			return
		}
		must(Prepare())
		readClose, err := client.SendVPNOperateCommand(kubeConfig, nameSpace, command.Reconnect, workloads)
		if err != nil {
			log.Warn(err)
			return
		}
		stream := bufio.NewReader(readClose)
		for {
			if line, _, err := stream.ReadLine(); errors.Is(err, io.EOF) {
				return
			} else {
				fmt.Println(string(line))
			}
		}
	},
}
