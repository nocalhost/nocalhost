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
	YamlCmd.AddCommand(yamlToJsonCmd)
}

var yamlToJsonCmd = &cobra.Command{
	Use:   "to-json",
	Short: "Convert yaml to json",
	Long:  `Convert yaml to json`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("fail to read from stdin: %v", err)
		}

		v := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(b, v); err != nil {
			log.Fatalf("fail to unmarshal from yaml: %v", err)
		}

		j, err := json.Marshal(convert(v))
		if err != nil {
			log.Fatalf("fail to marshal to json: %v", err)
		}

		fmt.Println(string(j))
	},
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
