/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

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
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	disconnectCmd.Flags().StringVar(&common.KubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	disconnectCmd.Flags().StringVarP(&common.NameSpace, "namespace", "n", "", "namespace")
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
		must(common.Prepare())
		readClose, err := client.SendVPNOperateCommand(common.KubeConfig, common.NameSpace, command.DisConnect, workloads)
		if err != nil {
			log.Warn(err)
			return
		}
		stream := bufio.NewReader(readClose)
		for {
			if line, _, err := stream.ReadLine(); errors.Is(err, io.EOF) {
				return
			} else {
				if len(line) == 0 {
					continue
				}
				if strings.Contains(string(line), util.EndSignOK) {
					readClose.Close()
					return
				} else if strings.Contains(string(line), util.EndSignFailed) {
					readClose.Close()
					return
				}
				fmt.Println(string(line))
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
				_ = os.Rename(filename, filepath.Join(os.TempDir(), "wintun.dll"))
			}
		}
	},
}
