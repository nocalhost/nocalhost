/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"fmt"
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
		if err := Prepare(); err != nil {
			log.FatalE(err, "")
		}

		rawKubeConfig, err := ioutil.ReadFile(kubeConfig)
		if err != nil {
			log.Fatalf("read %s fail, err %s \n", kubeConfig, err.Error())
		}

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
		if err != nil {
			log.Fatalf("init dep component fail, err: %s \n", err.Error())
		}
		client, err := clientgoutils.NewClientGoUtils(kubeConfig, app.DefaultInitWaitNameSpace)
		fmt.Printf("kubeconfig %s \n", kubeConfig)
		if err != nil || client == nil {
			log.Fatalf("new go client fail, err: %s, or check you kubeconfig\n", err)
		}
		// wait for job and deployment
		spinner := utils.NewSpinner(" waiting for nocalhost dep component ready, this will take a few minutes...")
		spinner.Start()
		// wait nocalhost-dep ready
		// max 5 min
		checkTime := 0
		for {
			isReady, _ := client.NameSpace(app.DefaultInitWaitNameSpace).CheckDeploymentReady(app.DefaultInitWaitDeployment)
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
		log.Info("nocalhost-dep has been installed, you can use `kubectl label namespace ${namespace} env=nocalhost` enable namespace dependency injection")
	},
}
