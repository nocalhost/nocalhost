package cmds

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	disconnectCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	disconnectCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	disconnectCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	disconnectCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(disconnectCmd)
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "disconnect",
	Long:  `disconnect`,
	PreRun: func(*cobra.Command, []string) {
		util.InitLogger(util.Debug)
	},
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(false)
		if err != nil {
			log.Warn(err)
			return
		}
		must(Prepare())
		readClose, err := client.SendVPNOperateCommand(kubeConfig, nameSpace, command.DisConnect, workloads)
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
				if strings.Contains(string(line), util.EndSignOK) {
					readClose.Close()
					return
				} else if strings.Contains(string(line), util.EndSignFailed) {
					readClose.Close()
					return
				}
			}
		}
	},
	PostRun: func(_ *cobra.Command, _ []string) {
		if util.IsWindows() && len(workloads) == 0 {
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
