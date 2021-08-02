/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	dbCompactCmd.Flags().StringVar(&appName, "app", "", "Leveldb data of specified application")
	dbCompactCmd.Flags().StringVar(&levelDbKey, "key", "", "The key of leveldb data")
	dbCmd.AddCommand(dbCompactCmd)
}

var dbCompactCmd = &cobra.Command{
	Use:   "compact",
	Short: "compact leveldb data",
	Long:  `compact leveldb data`,
	Run: func(cmd *cobra.Command, args []string) {
		must(nocalhost.CompactApplicationDb(nameSpace, appName, levelDbKey))
		log.Info("Db has been compacted")
	},
}
