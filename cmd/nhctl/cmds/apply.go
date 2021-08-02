/*Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
This source code is licensed under the Apache License Version 2.0.*/

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	rootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply [NAME] [MANIFEST]",
	Short: "Apply manifest",
	Long:  `Apply manifest`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.Errorf("%q requires at least 2 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		path := args[1]

		initApp(applicationName)
		manifests := clientgoutils.LoadValidManifest([]string{path})

		err := nocalhostApp.GetClient().Apply(
			manifests, false,
			app.StandardNocalhostMetas(nocalhostApp.Name, nocalhostApp.NameSpace), "",
		)
		if err != nil {
			log.Fatal(err)
		}
	},
}
