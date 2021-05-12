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
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io"
	"nocalhost/pkg/nhctl/log"
	"os"
)

func init() {
	YamlCmd.AddCommand(yamlFromJsonCmd)
}

var yamlFromJsonCmd = &cobra.Command{
	Use:   "from-json",
	Short: "Convert json to yaml",
	Long:  `Convert json to yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("fail to read from stdin: %v", err)
		}

		v := make(map[string]interface{})
		if err := json.Unmarshal(b, &v); err != nil {
			log.Fatalf("fail to unmarshal from json: %v", err)
		}

		y, err := yaml.Marshal(v)
		if err != nil {
			log.Fatalf("fail to marshal to yaml: %v", err)
		}

		fmt.Println(string(y))
	},
}
