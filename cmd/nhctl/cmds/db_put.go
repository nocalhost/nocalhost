/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	dbPutCmd.Flags().StringVar(&appName, "app", "", "Leveldb data of specified application")
	dbPutCmd.Flags().StringVar(&levelDbKey, "key", "", "The key of leveldb data")
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
			must(nocalhost.UpdateKey(nameSpace, appName, levelDbKey, levelDbValue))
		} else if levelDbValueFile != "" {
			bys, err := ioutil.ReadFile(levelDbValueFile)
			must(errors.Wrap(err, ""))
			must(nocalhost.UpdateKey(nameSpace, appName, levelDbKey, string(bys)))
		} else {
			log.Fatal("--value or --file must be specified")
		}
	},
}
