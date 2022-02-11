/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset application",
	Long:  `reset application`,
	Run: func(cmd *cobra.Command, args []string) {

		must(common.Prepare())

		if len(args) > 0 {
			applicationName := args[0]
			if applicationName != "" {
				resetApplication(applicationName)
				return
			}
		}

		// Reset all applications under specify namespace
		metas, err := nocalhost.GetApplicationMetas(common.NameSpace, common.KubeConfig)
		mustI(err, "Failed to get applications")
		for _, meta := range metas {
			resetApplication(meta.Application)
		}
	},
}

func resetApplication(applicationName string) {
	var err error
	nocalhostApp, err := common.InitApp(applicationName)
	must(err)

	// Stop BackGroup Process
	appProfile, _ := nocalhostApp.GetProfile()
	for _, profile := range appProfile.SvcProfile {
		nhSvc, err := nocalhostApp.InitService(profile.GetName(), profile.GetType())
		must(err)
		if nhSvc.IsInDevMode() {
			utils.Should(nhSvc.StopSyncAndPortForwardProcess(true))
		} else if len(profile.DevPortForwardList) > 0 {
			utils.Should(nhSvc.StopAllPortForward())
		}
	}

	// Remove files
	time.Sleep(1 * time.Second)
	if err = nocalhost.CleanupAppFilesUnderNs(common.NameSpace, nocalhostApp.GetAppMeta().NamespaceId); err != nil {
		log.WarnE(err, "")
	} else {
		log.Info("Files have been clean up")
	}
	log.Infof("Application %s has been reset.\n", applicationName)
}
