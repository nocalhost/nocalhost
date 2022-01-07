/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/nocalhost"
)

func init() {
	dbAllCmd.Flags().StringVar(&appName, "app", "", "List leveldb data of specified application")
	dbAllCmd.Flags().StringVar(&nid, "nid", "", "Nid of namespace")
	//pvcListCmd.Flags().StringVar(&pvcFlags.Svc, "controller", "", "List PVCs of specified service")
	dbCmd.AddCommand(dbAllCmd)
}

var dbAllCmd = &cobra.Command{
	Use:   "all [NAME]",
	Short: "Get all leveldb data",
	Long:  `Get all leveldb data`,
	Run: func(cmd *cobra.Command, args []string) {
		result, err := nocalhost.ListAllFromApplicationDb(common.NameSpace, appName, nid)
		must(err)
		for key, val := range result {
			fmt.Printf("%s=%s\n", key, val)
		}
	},
}
