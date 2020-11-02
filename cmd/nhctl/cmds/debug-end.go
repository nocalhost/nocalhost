package cmds

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"sort"
	"strings"
)

func init() {
	debugEndCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	debugEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	debugEndCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	debugCmd.AddCommand(debugEndCmd)
}

var debugEndCmd = &cobra.Command{
	Use:   "end",
	Short: "end debug model",
	Long: `end debug model`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		if deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}
		fmt.Println("end debug...")
		// end file sync
		fmt.Println("ending file sync...")
		EndFileSync()

		fmt.Println("stopping port-forward...")
		StopPortForward()

		fmt.Println("roll back deployment...")
		DeploymentRollBackToPreviousRevision()
	},
}

func StopPortForward(){


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

	_, err = tools.ExecCommand(nil,true, "kill", "-1", pid)
	if err != nil {
		printlnErr("failed to stop port forward",err)
		return
	} else {
		fmt.Println("port-forward stopped.")
	}
}

func EndFileSync() {
	output , _ := tools.ExecCommand(nil, false ,"mutagen", "sync", "list")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Name") {
			strs := strings.Split(line, ":")
			if len(strs) >= 2 {
				sessionName := strings.TrimLeft(strs[1]," ")
				fmt.Printf("terminate sync session :%s \n", sessionName)
				_, err := tools.ExecCommand(nil,true,"mutagen", "sync", "terminate", sessionName)
				if err != nil {
					printlnErr("failed to terminate sync session", err)
				}else {
					// todo confirm session's status
					fmt.Println("sync session has been terminated.")
				}

			}
			//fmt.Println(line)
		}
	}
}

func DeploymentRollBackToPreviousRevision(){
	deploymentsClient, err := GetDeploymentClient(nameSpace)
	if err != nil {
		fmt.Printf("%v",err)
		return
	}

	dep, err := deploymentsClient.Get(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", dep.Name, err)
		return
	}

	fmt.Printf("rolling deployment back to previous revision\n")
	rss, err := GetReplicaSetsControlledByDeployment(deployment)
	if err != nil {
		fmt.Printf("failed to get rs list, err:%v\n", err)
		return
	}
	// find previous replicaSet
	if len(rss) < 2 {
		fmt.Println("no history to roll back")
		return
	}

	keys := make([]int,0)
	for rs := range rss {
		keys = append(keys, rs)
	}
	sort.Ints(keys)

	dep.Spec.Template = rss[keys[len(keys)-2]].Spec.Template // previous replicaSet is the second largest revision number : keys[len(keys)-2]
	_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println("failed rolling back")
	} else {
		fmt.Println("rolling back!")
	}
	// todo wait util rollback completed

}

