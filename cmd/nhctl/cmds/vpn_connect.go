package cmds

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
)

var workloads string

func init() {
	connectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	connectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	connectCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
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
		readClose, err := client.SendVPNOperateCommand(kubeConfig, nameSpace, command.Connect, workloads)
		if err != nil {
			log.Fatal(err)
			return
		}
		stream := bufio.NewReader(readClose)
		for {
			if line, _, err := stream.ReadLine(); errors.Is(err, io.EOF) {
				return
			} else {
				fmt.Println(string(line))
				if strings.Contains(string(line), util.EndSignOK) {
					readClose.Close()
					return
				} else if strings.Contains(string(line), util.EndSingleFailed) {
					readClose.Close()
					return
				}
			}
		}
	},
}
