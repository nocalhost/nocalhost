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
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"sort"
	"strings"
)

func init() {
	devEndCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	devEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	debugCmd.AddCommand(devEndCmd)
}

var devEndCmd = &cobra.Command{
	Use:   "end",
	Short: "end dev model",
	Long:  `end dev model`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		if deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}
		fmt.Println("exiting dev model...")
		// end file sync
		fmt.Println("ending file sync...")
		EndFileSync()

		fmt.Println("stopping port-forward...")
		StopPortForward()

		fmt.Println("roll back deployment...")
		DeploymentRollBackToPreviousRevision()
	},
}

func StopPortForward() {

	_, err := os.Stat(".pid")
	var bys []byte
	if err == nil {
		bys, err = ioutil.ReadFile(".pid")
	}
	if err != nil {
		printlnErr("failed to get pid", err)
		return
	}

	pid := string(bys)

	_, err = tools.ExecCommand(nil, true, "kill", "-1", pid)
	if err != nil {
		printlnErr("failed to stop port forward", err)
		return
	} else {
		fmt.Println("port-forward stopped.")
	}
}

func EndFileSync() {
	output, _ := tools.ExecCommand(nil, false, "mutagen", "sync", "list")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Name") {
			strs := strings.Split(line, ":")
			if len(strs) >= 2 {
				sessionName := strings.TrimLeft(strs[1], " ")
				fmt.Printf("terminate sync session :%s \n", sessionName)
				_, err := tools.ExecCommand(nil, true, "mutagen", "sync", "terminate", sessionName)
				if err != nil {
					printlnErr("failed to terminate sync session", err)
				} else {
					// todo confirm session's status
					fmt.Println("sync session has been terminated.")
				}
			}
		}
	}
}

func DeploymentRollBackToPreviousRevision() {

	clientUtils, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
	clientgoutils.Must(err)

	dep, err := clientUtils.GetDeployment(context.TODO(), nameSpace, deployment)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", dep.Name, err)
		return
	}

	fmt.Printf("rolling deployment back to previous revision\n")
	rss, err := clientUtils.GetReplicaSetsControlledByDeployment(context.TODO(), nameSpace, deployment)
	if err != nil {
		fmt.Printf("failed to get rs list, err:%v\n", err)
		return
	}
	// find previous replicaSet
	if len(rss) < 2 {
		fmt.Println("no history to roll back")
		return
	}

	keys := make([]int, 0)
	for rs := range rss {
		keys = append(keys, rs)
	}
	sort.Ints(keys)

	dep.Spec.Template = rss[keys[len(keys)-2]].Spec.Template // previous replicaSet is the second largest revision number : keys[len(keys)-2]
	//_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	_, err = clientUtils.UpdateDeployment(context.TODO(), nameSpace, dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Println("failed rolling back")
	} else {
		fmt.Println("rolling back!")
	}

}
