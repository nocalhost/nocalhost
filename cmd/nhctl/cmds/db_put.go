/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	dbPutCmd.Flags().StringVar(&appName, "app", "", "Leveldb data of specified application")
	dbPutCmd.Flags().StringVar(&levelDbKey, "key", "", "The key of leveldb data")
	dbPutCmd.Flags().StringVar(&nid, "nid", "", "Nid of namespace")
	dbPutCmd.Flags().StringVar(&levelDbValue, "value", "", "The value of leveldb data")
	dbPutCmd.Flags().StringVarP(&levelDbValueFile, "file", "f", "", "The value of leveldb data")
	dbCmd.AddCommand(dbPutCmd)
}

var dbPutCmd = &cobra.Command{
	Use:   "put",
	Short: "update leveldb data",
	Long:  `update leveldb data`,
	Run: func(cmd *cobra.Command, args []string) {

		if levelDbKey == "" {
			log.Fatal("--key must be specified")
		}

		if levelDbValue != "" {
			must(nocalhost.UpdateKey(common.NameSpace, appName, nid, levelDbKey, levelDbValue))
		} else if levelDbValueFile != "" {
			bys, err := ioutil.ReadFile(levelDbValueFile)
			must(errors.Wrap(err, ""))
			must(nocalhost.UpdateKey(common.NameSpace, appName, nid, levelDbKey, string(bys)))
		} else {
			log.Fatal("--value or --file must be specified")
		}
	},
}
