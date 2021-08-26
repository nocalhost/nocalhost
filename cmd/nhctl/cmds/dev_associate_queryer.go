/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/json"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

var current bool
var excludeStatus []string
var jsonOutput bool

func init() {
	devAssociateQueryerCmd.Flags().StringVarP(&workDir, "associate", "s", "", "dev mode work directory")
	devAssociateQueryerCmd.Flags().BoolVar(
		&current, "current", false, "show the active svc most recently associate to the path",
	)
	devAssociateQueryerCmd.Flags().StringArrayVarP(
		&excludeStatus, "exclude-status", "e", []string{},
		"exclude some sync status, value: [disconnected, outOfSync, scanning, syncing, error, idle, end]",
	)
	devAssociateQueryerCmd.Flags().BoolVar(
		&jsonOutput, "json", false,
		"return the json format of results",
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
		if current {

			pack, err := devPath.GetDefaultPack()
			if err == dev_dir.NO_DEFAULT_PACK {
				println("{}")
				return
			} else {
				must(err)
				printing(genAssociateSvcPack(pack))
				return
			}
		} else {

			set := make(map[string]string, 0)
			for _, status := range excludeStatus {
				set[status] = ""
			}

			allPacks := devPath.GetAllPacks()

			asps := make([]*AssociateSvcPack, 0)
			for _, pack := range allPacks.Packs {
				asp := genAssociateSvcPack(pack)
				if _, exclude := set[string(asp.SyncthingStatus.Status)]; exclude {
					continue
				}

				asps = append(asps, asp)
			}

			printing(asps)
		}
	},
}

func printing(output interface{}) {
	if jsonOutput {
		marshal, err := json.Marshal(output)
		must(err)
		fmt.Printf(string(marshal))
	} else {
		marshal, err := yaml.Marshal(output)
		must(err)
		fmt.Printf(string(marshal))
	}
}

func genAssociateSvcPack(pack *dev_dir.SvcPack) *AssociateSvcPack {
	nhctlPath, _ := utils.GetNhctlPath()
	kubeconfigPath := nocalhost.GetOrGenKubeConfigPath(pack.GetKubeConfigBytes())

	asp := &AssociateSvcPack{
		pack,
		utils.Sha1ToString(string(pack.Key())),
		kubeconfigPath,
		fmt.Sprintf(
			"%s sync-status %s --namespace %s --deployment %s --controller-type %s --kubeconfig %s",
			nhctlPath, pack.App, pack.Ns, pack.Svc, pack.SvcType, kubeconfigPath,
		),
		SyncStatus(nil, pack.Ns, pack.App, pack.Svc, pack.SvcType.String(), kubeconfigPath),
	}
	return asp
}

type AssociateSvcPack struct {
	*dev_dir.SvcPack     `yaml:"svc_pack" json:"svc_pack"`
	Sha                  string `yaml:"sha" json:"sha"`
	KubeconfigPath       string `yaml:"kubeconfig_path" json:"kubeconfig_path"`
	SyncStatusCmd        string `yaml:"sync_status_cmd" json:"sync_status_cmd"`
	*req.SyncthingStatus `yaml:"syncthing_status" json:"syncthing_status"`
}
