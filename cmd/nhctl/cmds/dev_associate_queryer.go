/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

var all bool

func init() {
	devAssociateQueryerCmd.Flags().StringVarP(&workDir, "associate", "s", "", "dev mode work directory")
	devAssociateQueryerCmd.Flags().BoolVar(
		&all, "all", false, "show all svc associate to the path",
	)
	debugCmd.AddCommand(devAssociateQueryerCmd)
}

var devAssociateQueryerCmd = &cobra.Command{
	Use:   "associate-queryer [Name]",
	Short: "associate dev dir queryer",
	Long:  "associate dev dir queryer",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if workDir == "" {
			log.Fatal("associate must specify")
			return
		}

		devPath := dev_dir.DevPath(workDir)
		if all {
			packs := devPath.GetAllPacks()
			marshal, _ := yaml.Marshal(packs)
			println(string(marshal))
			return
		}

		pack, err := devPath.GetDefaultPack()
		must(err)
		nhctlPath, _ := utils.GetNhctlPath()
		kubeconfigPath := nocalhost.GetOrGenKubeConfigPath(pack.GetKubeConfigBytes())

		asp := &AssociateSvcPack{
			pack,
			kubeconfigPath,
			fmt.Sprintf(
				"%s sync-status %s --namespace %s --deployment %s --controller-type %s --kubeconfig %s",
				nhctlPath, pack.App, pack.Ns, pack.Svc, pack.SvcType, kubeconfigPath,
			),
		}
		pack.GetKubeConfigBytes()

		marshal, err := yaml.Marshal(asp)
		must(err)

		fmt.Printf(string(marshal))
	},
}

type AssociateSvcPack struct {
	*dev_dir.SvcPack `yaml:"svc_pack"`
	KubeconfigPath   string `yaml:"kubeconfig_path"`
	SyncStatusCmd    string `yaml:"sync_status_cmd"`
}