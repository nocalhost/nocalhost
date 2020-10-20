package cmd

import (
	"context"
	"fmt"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"os/user"
	"strconv"
)

func getClientSet() (*kubernetes.Clientset, error) {
	home := GetHomePath()
	kubeconfigPath := fmt.Sprintf("%s/.kube/config", home)  // default kubeconfig
	if kubeconfig != "" {
		//if strings.HasPrefix(kubeconfig, "/") {
		//	kubeconfigPath = kubeconfig
		//} else {
		//
		//}
		kubeconfigPath = kubeconfig
	}

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("%v",err)
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Printf("%v",err)
		return nil, err
	}
	return clientSet, nil
}

func GetDeploymentClient(nameSpace string) (appsV1.DeploymentInterface, error){
	clientSet, err := getClientSet()
	if err != nil {
		fmt.Printf("%v",err)
		return nil , err
	}

	deploymentsClient := clientSet.AppsV1().Deployments(nameSpace)
	return deploymentsClient, nil
}

func GetPodClient(nameSpace string) (coreV1.PodInterface, error){
	clientSet, err := getClientSet()
	if err != nil {
		fmt.Printf("%v",err)
		return nil , err
	}

	podClient := clientSet.CoreV1().Pods(nameSpace)
	return podClient, nil
}

// revision:ReplicaSet
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

func printlnErr(info string, err error) {
	fmt.Printf("%s, err: %v\n", info, err)
}


func GetHomePath() string {
	u , err := user.Current()
	if err == nil {
		return u.HomeDir
	}
	return ""
}

