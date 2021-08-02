/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"time"
)

func init() {
	InitCmd.AddCommand(InitDepCommand)
}

var InitDepCommand = &cobra.Command{
	Use:   "dep",
	Short: "dep component",
	Long:  `dep component`,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		must(Prepare())

		rawKubeConfig, err := ioutil.ReadFile(kubeConfig)
		must(errors.Wrap(err, ""))

		goClient, err := clientgo.NewAdminGoClient(rawKubeConfig)
		if err != nil || goClient == nil {
			log.Fatalf("create go client fail, err: %s, or check you kubeconfig\n", err)
		}
		clusterSetUp := setupcluster.NewSetUpCluster(goClient)
		tag := Version
		if Branch != app.DefaultNocalhostMainBranch {
			tag = DevGitCommit
		}
		_, err, _ = clusterSetUp.InitCluster(tag)
		mustI(err, "init dep component fail")

		client, err := clientgoutils.NewClientGoUtils(kubeConfig, app.DefaultInitWaitNameSpace)
		fmt.Printf("kubeconfig %s \n", kubeConfig)
		if err != nil || client == nil {
			log.Fatalf("new go client fail, err: %s, or check you kubeconfig\n", err)
			return
		}
		// wait for job and deployment
		spinner := utils.NewSpinner(
			" waiting for nocalhost dep component ready," +
				" this will take a few minutes...",
		)
		spinner.Start()
		// wait nocalhost-dep ready
		// max 5 min
		checkTime := 0
		for {
			isReady, _ := client.NameSpace(app.DefaultInitWaitNameSpace).
				CheckDeploymentReady(app.DefaultInitWaitDeployment)
			if isReady {
				break
			}
			checkTime = checkTime + 1
			if checkTime > 1500 {
				break
			}
			time.Sleep(time.Duration(200) * time.Millisecond)
		}
		spinner.Stop()
		log.Info(
			"nocalhost-dep has been installed, you can use `kubectl " +
				"label namespace ${namespace} env=nocalhost` enable namespace dependency injection",
		)
	},
}
