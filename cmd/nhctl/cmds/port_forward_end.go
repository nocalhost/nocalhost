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
	"nocalhost/pkg/nhctl/log"
)

var portForwardEndOptions = &app.PortForwardEndOptions{}

func init() {
	portForwardEndCmd.Flags().StringVarP(&common.WorkloadName, "deployment", "d", "", "k8s deployment which you want to forward to")
	portForwardEndCmd.Flags().StringVarP(&portForwardEndOptions.Port, "port", "p", "", "stop specify port-forward")
	portForwardEndCmd.Flags().StringVarP(&common.ServiceType, "type", "t", "deployment", "specify service type")
	PortForwardCmd.AddCommand(portForwardEndCmd)
}

var portForwardEndCmd = &cobra.Command{
	Use:   "end [NAME]",
	Short: "stop port-forward",
	Long:  `stop port-forward`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
		must(err)
		err = nocalhostSvc.StopPortForwardByPort(portForwardEndOptions.Port)
		if err != nil {
			log.WarnE(err, "stop port-forward fail")
		} else {
			log.Infof("%s port-forward has been stop", portForwardEndOptions.Port)
		}
		// notify daemon to invalid cache before return
		if client, err := daemon_client.GetDaemonClient(false); err == nil {
			_ = client.SendFlushDirMappingCacheCommand(
				nocalhostSvc.NameSpace, nocalhostSvc.AppMeta.NamespaceId, applicationName,
			)
		}
	},
}
