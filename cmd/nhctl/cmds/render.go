/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/log"
	customyaml3 "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
)

var renderOps = &RenderOps{}

type RenderOps struct {
	envPath string
	origin  bool
}

func init() {
	renderCmd.Flags().StringVarP(&renderOps.envPath, "env path", "e", "", "the env file for render injection")
	renderCmd.Flags().BoolVar(&renderOps.origin, "origin", false, "return the origin result after rendered")
	rootCmd.AddCommand(renderCmd)
}

var renderCmd = &cobra.Command{
	Use: "render [NAME]",
	Short: "Render the file for debugging",
	Long: `Render the file for debugging`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		render, err := envsubst.Render(
			envsubst.LocalFileRenderItem{FilePathEnhance: fp.NewFilePath(args[0])},
			fp.NewFilePath(renderOps.envPath),
		)
		must(errors.Wrap(err, ""))

		if renderOps.origin {
			fmt.Print(render)
		} else {
			m := map[string]interface{}{}
			_ = customyaml3.Unmarshal([]byte(render), m)

			if len(m) == 0 {
				err := yaml.Unmarshal([]byte(render), m)
				if err != nil {
					log.Fatalf(
						"%s\n\n======\n\nRender Error: %s, Please check the render result above", render, err.Error(),
					)
				}
			} else {
				marshal, _ := customyaml3.Marshal(m)
				fmt.Print(string(marshal))
			}
		}
	},
}
