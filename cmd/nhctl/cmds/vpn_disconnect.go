package cmds

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
	"os"
	"path/filepath"
)

var disconnect pkg.ConnectOptions

func init() {
	disconnectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	disconnectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	disconnectCmd.PersistentFlags().StringArrayVar(&disconnect.Workloads, "workloads", []string{}, "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	disconnectCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	VPNCmd.AddCommand(disconnectCmd)
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "disconnect",
	Long:  `disconnect`,
	PreRun: func(*cobra.Command, []string) {
		util.InitLogger(util.Debug)
	},
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(true)
		if err != nil {
			log.Warn(err)
			return
		}
		must(Prepare())
		err = client.SendVPNOperateCommand(kubeConfig, nameSpace, command.DisConnect, workloads)
		if err != nil {
			log.Warn(err)
		}
	},
	PostRun: func(_ *cobra.Command, _ []string) {
		if util.IsWindows() {
			if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
				return err != nil
			}, func() error {
				return driver.UninstallWireGuardTunDriver()
			}); err != nil {
				wd, _ := os.Getwd()
				filename := filepath.Join(wd, "wintun.dll")
				if err = os.Rename(filename, filepath.Join(os.TempDir(), "wintun.dll")); err != nil {
					log.Warn(err)
				}
			}
		}
	},
}
