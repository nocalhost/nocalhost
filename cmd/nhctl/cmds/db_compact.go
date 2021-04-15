/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
