/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/coloredoutput"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
)

func init() {
	devEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devEndCmd.Flags().StringVarP(&serviceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	debugCmd.AddCommand(devEndCmd)
}

var devEndCmd = &cobra.Command{
	Use:   "end [NAME]",
	Short: "end dev model",
	Long:  `end dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if !nocalhostSvc.IsInReplaceDevMode() && !nocalhostSvc.IsInDuplicateDevMode() {
			log.Fatalf("Service %s is not in DevMode", deployment)
		}

		var needToRecoverHPA bool
		if !nocalhostSvc.IsInDuplicateDevMode() {
			needToRecoverHPA = true
		}

		must(nocalhostSvc.DevEnd(false))
		utils.Should(nocalhostSvc.DecreaseDevModeCount())

		// Recover hpa
		if needToRecoverHPA {
			log.Info("Recovering HPA...")
			hl, err := nocalhostSvc.ListHPA()
			if err != nil {
				log.WarnE(err, "Failed to find HPA")
			}
			if len(hl) == 0 {
				log.Info("No HPA found")
			}
			for _, h := range hl {
				if len(h.Annotations) == 0 {
					continue
				}
				if max, ok := h.Annotations[_const.HPAOriginalMaxReplicasKey]; ok {
					maxInt, err := strconv.ParseInt(max, 0, 0)
					if err != nil {
						log.WarnE(err, "")
						continue
					}
					h.Spec.MaxReplicas = int32(maxInt)
				}
				if min, ok := h.Annotations[_const.HPAOriginalMinReplicasKey]; ok {
					minInt, err := strconv.ParseInt(min, 0, 0)
					if err != nil {
						log.WarnE(err, "")
						continue
					}
					minInt32 := int32(minInt)
					h.Spec.MinReplicas = &minInt32
				}
				if _, err = nocalhostSvc.Client.UpdateHPA(&h); err != nil {
					log.WarnE(err, fmt.Sprintf("Failed to update hpa %s", h.Name))
				} else {
					log.Infof("HPA %s has been recovered", h.Name)
				}
			}
		}
		//fmt.Println()
		coloredoutput.Success("DevMode has been ended")
	},
}
