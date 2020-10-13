package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"strconv"
)

func init() {
	debugEndCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	debugEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	//debugEndCmd.Flags().StringVarP(&lang, "type", "l", "", "the development language, eg: java go python")
	//debugEndCmd.Flags().StringVarP(&image, "image", "i", "", "image of development container")
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
		DeploymentRollBackToPreviousRevision()
	},
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
	//for _, key := range keys {
	//	fmt.Printf("%d %s\n", key, rss[key].Name)
	//}

	fmt.Println(rss[keys[len(keys)-2]].Name)
	dep.Spec.Template = rss[keys[len(keys)-2]].Spec.Template // previous replicaSet is the second largest revision number : keys[len(keys)-2]
	_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println("failed rolling back")
	}else {
		fmt.Println("rolling back!")
	}

}

func GetReplicaSetsControlledByDeployment(deployment string) (map[int]*v1.ReplicaSet,error) {
	var rsList *v1.ReplicaSetList
	clientSet, err := getClientSet()
	if err == nil {
		replicaSetsClient := clientSet.AppsV1().ReplicaSets(nameSpace)
		rsList, err = replicaSetsClient.List(context.TODO(),metav1.ListOptions{})
	}
	if err != nil {
		fmt.Printf("failed to get rs: %v\n", err)
		return nil,err
	}

	rsMap := make(map[int]*v1.ReplicaSet)
	for _, item := range rsList.Items {
		if item.OwnerReferences != nil {
			for _, owner := range item.OwnerReferences {
				if owner.Name == deployment && item.Annotations["deployment.kubernetes.io/revision"] != "" {
					fmt.Printf("%s %s\n", item.Name, item.Annotations["deployment.kubernetes.io/revision"] )
					revision, err := strconv.Atoi(item.Annotations["deployment.kubernetes.io/revision"])
					if err == nil {
						rsMap[revision] = item.DeepCopy()
					}
				}
			}
		}
	}
	return rsMap, nil
}