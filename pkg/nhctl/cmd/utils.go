package cmd

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func getClientSet() (*kubernetes.Clientset, error) {
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", "/Users/xinxinhuang/.kube/config")
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

func GetDeploymentClient(nameSpace string) (v1.DeploymentInterface, error){
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
