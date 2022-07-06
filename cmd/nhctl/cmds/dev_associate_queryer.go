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
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/internal/nhctl/utils"
	k8sutil "nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"sort"
)

var current bool
var excludeStatus []string
var jsonOutput bool

func init() {
	devAssociateQueryerCmd.Flags().StringVarP(
		&workDir,
		"local-sync", "s", "",
		"the local directory synchronized to the remote container under dev mode",
	)
	devAssociateQueryerCmd.Flags().StringVar(
		&workDirDeprecated, "associate", "",
		"the local directory synchronized to the remote container under dev mode(deprecated)",
	)
	devAssociateQueryerCmd.Flags().BoolVar(
		&current, "current", false,
		"show the active svc most recently associate to the path",
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
		if workDir == "" && workDirDeprecated == "" {
			log.Fatal("--local-sync must specify")
			return
		}

		if workDirDeprecated != "" {
			workDir = workDirDeprecated
		}

		devPath := dev_dir.DevPath(workDir)
		if current {

			pack, err := devPath.GetDefaultPack()
			if err == dev_dir.NO_DEFAULT_PACK {
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

			// if a svc pack has a container
			// then remove the svc with same define(without container)
			uniqueMapForRemoveNoneContainerSvcPack := map[string]string{}

			for _, pack := range allPacks.Packs {
				asp := genAssociateSvcPack(pack)
				if _, exclude := set[string(asp.SyncthingStatus.Status)]; exclude {
					continue
				}

				asps = append(asps, asp)

				if asp.Container != "" {
					uniqueMapForRemoveNoneContainerSvcPack[string(asp.KeyWithoutContainer())] = ""
				}
			}

		loop:
			for index, currentAsp := range asps {
				if _, ok := uniqueMapForRemoveNoneContainerSvcPack[string(currentAsp.Key())]; ok {
					asps = append(asps[:index], asps[index+1:]...)
					goto loop
				}
			}

			sort.Slice(
				asps, func(i, j int) bool {
					return asps[i].Sha > asps[j].Sha
				},
			)

			printing(asps)
		}
	},
}

func printing(output interface{}) {
	if jsonOutput {
		marshal, err := json.Marshal(output)
		must(err)
		fmt.Print(string(marshal))
	} else {
		marshal, err := yaml.Marshal(output)
		must(err)
		fmt.Print(string(marshal))
	}
}

func genAssociateSvcPack(pack *dev_dir.SvcPack) *AssociateSvcPack {
	nhctlPath, _ := utils.GetNhctlPath()
	kubeConfigBytes, server := pack.GetKubeConfigBytesAndServer()
	kubeConfigPath := k8sutil.GetOrGenKubeConfigPath(kubeConfigBytes)

	asp := &AssociateSvcPack{
		pack,
		utils.Sha1ToString(string(pack.Key())),
		kubeConfigPath,
		server,
		fmt.Sprintf(
			"%s sync-status %s --namespace %s --deployment %s --controller-type %s --kubeconfig %s",
			nhctlPath, pack.App, pack.Ns, pack.Svc, pack.SvcType, kubeConfigPath,
		),
		SyncStatus(nil, pack.Ns, pack.App, pack.Svc, pack.SvcType.String(), kubeConfigPath),
	}
	return asp
}

type AssociateSvcPack struct {
	*dev_dir.SvcPack     `yaml:"svc_pack" json:"svc_pack"`
	Sha                  string `yaml:"sha" json:"sha"`
	KubeconfigPath       string `yaml:"kubeconfig_path" json:"kubeconfig_path"`
	Server               string `yaml:"server" json:"server"`
	SyncStatusCmd        string `yaml:"sync_status_cmd" json:"sync_status_cmd"`
	*req.SyncthingStatus `yaml:"syncthing_status" json:"syncthing_status"`
}
