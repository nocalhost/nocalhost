/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"nocalhost/cmd/nhctl/cmds/common"
)

func init() {
	portForwardListCmd.Flags().BoolVar(&listFlags.Yaml, "yaml", false, "use yaml as out put")
	portForwardListCmd.Flags().BoolVar(&listFlags.Json, "json", false, "use json as out put")
	PortForwardCmd.AddCommand(portForwardListCmd)
}

type PortForwardItem struct {
	SvcName         string `json:"svcName" yaml:"svcName"`
	ServiceType     string `json:"servicetype" yaml:"servicetype"`
	Port            string `json:"port" yaml:"port"`
	Status          string `json:"status" yaml:"status"`
	Role            string `json:"role" yaml:"role"`
	Sudo            bool   `json:"sudo" yaml:"sudo"`
	DaemonServerPid int    `json:"daemonserverpid" yaml:"daemonserverpid"`
	Updated         string `json:"updated" yaml:"updated"`
	Reason          string `json:"reason" yaml:"reason"`
}

var portForwardListCmd = &cobra.Command{
	Use:   "list [NAME]",
	Short: "list port-forward",
	Long:  `list port-forward`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		nocalhostApp, err := common.InitApp(applicationName)
		must(err)

		p, err := nocalhostApp.GetProfile()
		must(err)

		pfList := make([]PortForwardItem, 0)
		for _, sp := range p.SvcProfile {
			for _, pf := range sp.DevPortForwardList {
				pfList = append(pfList, PortForwardItem{
					SvcName:         sp.GetName(),
					ServiceType:     sp.GetType(),
					Port:            fmt.Sprintf("%d:%d", pf.LocalPort, pf.RemotePort),
					Status:          pf.Status,
					Role:            pf.Role,
					Sudo:            pf.Sudo,
					DaemonServerPid: pf.DaemonServerPid,
					Updated:         pf.Updated,
					Reason:          pf.Reason,
				})
			}
		}

		var bys []byte
		if listFlags.Json {
			bys, err = json.Marshal(pfList)
			must(err)
		}

		if listFlags.Yaml {
			bys, err = yaml.Marshal(pfList)
			must(err)
		}
		fmt.Printf("%s", string(bys))

	},
}
