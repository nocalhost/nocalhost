/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

var nid string

func init() {
	dbSizeCmd.Flags().StringVar(&appName, "app", "", "List leveldb data of specified application")
	dbSizeCmd.Flags().StringVar(&nid, "nid", "", "Nid of namespace")
	//pvcListCmd.Flags().StringVar(&pvcFlags.Svc, "controller", "", "List PVCs of specified service")
	dbCmd.AddCommand(dbSizeCmd)
}

var dbSizeCmd = &cobra.Command{
	Use:   "size [NAME]",
	Short: "Get all leveldb data",
	Long:  `Get all leveldb data`,
	Run: func(cmd *cobra.Command, args []string) {
		size, err := nocalhost.GetApplicationDbSize(nameSpace, appName, nid)
		must(err)
		log.Info(size)
	},
}
