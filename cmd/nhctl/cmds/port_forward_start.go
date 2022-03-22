/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

var portForwardOptions = &app.PortForwardOptions{}

func init() {
	portForwardStartCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", "", "k8s deployment which you want to forward to",
	)
	portForwardStartCmd.Flags().StringSliceVarP(
		&portForwardOptions.DevPort, "dev-port", "p", []string{},
		"port-forward between pod and local, such 8080:8080 or :8080(random localPort)",
	)
	//portForwardStartCmd.Flags().BoolVarP(&portForwardOptions.RunAsDaemon,
	// "daemon", "m", true, "if port-forward run as daemon")
	portForwardStartCmd.Flags().BoolVarP(
		&portForwardOptions.Forward, "forward", "f", false,
		"forward actually, deprecated",
	)
	portForwardStartCmd.Flags().StringVarP(
		&portForwardOptions.PodName, "pod", "", "",
		"specify pod name",
	)
	portForwardStartCmd.Flags().StringVarP(
		&container, "container", "c", "",
		"which container of pod to run command",
	)
	portForwardStartCmd.Flags().StringVarP(
		&common.ServiceType, "type", "t", "deployment",
		"specify service type",
	)
	portForwardStartCmd.Flags().StringVarP(
		&portForwardOptions.Way, "way", "", "manual",
		"specify port-forward way, deprecated",
	)
	portForwardStartCmd.Flags().BoolVarP(
		&portForwardOptions.Follow, "follow", "", false,
		"stock here waiting for disconnect or return immediately",
	)
	PortForwardCmd.AddCommand(portForwardStartCmd)
}

var portForwardStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "Forward local port to remote pod's port",
	Long:  `Forward local port to remote pod's port`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		applicationName := args[0]
		nocalhostApp, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
		must(err)

		log.Info("Starting port-forwarding")

		// find deployment pods
		podName, err := nocalhostSvc.GetDevModePodName()
		if err != nil {
			// use serviceType get pods name
			// can not find devContainer, means need port-forward normal service, get pods from command flags
			podName = portForwardOptions.PodName
		}

		var localPorts, remotePorts []int
		for _, port := range portForwardOptions.DevPort {
			localPort, remotePort, err := utils.GetPortForwardForString(port)
			if err != nil {
				log.WarnE(err, "")
				continue
			}
			localPorts = append(localPorts, localPort)
			remotePorts = append(remotePorts, remotePort)
		}

		for index, localPort := range localPorts {
			if portForwardOptions.Follow {
				must(nocalhostApp.PortForwardFollow(podName, localPort, remotePorts[index], nil))
			} else {
				must(nocalhostSvc.PortForward(podName, localPort, remotePorts[index], ""))
			}
		}
		// notify daemon to invalid cache before return
		if client, err := daemon_client.GetDaemonClient(false); err == nil {
			_ = client.SendFlushDirMappingCacheCommand(
				nocalhostApp.NameSpace, nocalhostApp.GetAppMeta().NamespaceId, applicationName,
			)
		}
	},
}
