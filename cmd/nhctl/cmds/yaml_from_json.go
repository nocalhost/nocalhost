/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
