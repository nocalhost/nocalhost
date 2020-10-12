package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//var localFolderName, remoteFolder, sshPort string
var nameSpace string

func init() {
	//install k8s
	debugCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	debugCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")

	//fileSyncCmd.Flags().StringVarP(&localFolderName, "local-folder", "l", "", "local folder path")
	//fileSyncCmd.Flags().StringVarP(&remoteFolder, "remote-folder", "r", "/home/code", "remote folder path")
	//fileSyncCmd.Flags().StringVarP(&sshPort, "port", "p", "22", "ssh port")
	//fileSyncCmd.Flags().StringVarP(&remoteFolder, "ssh passwd", "p", "", "ssh passwd")
	rootCmd.AddCommand(debugCmd)
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "enter debug model",
	Long: `enter debug model`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		//if remoteFolder == "" {
		//	fmt.Println("error: please use -r to specify a remote folder")
		//	return
		//}
		//TO-DO
		fmt.Println("enter debug...")
		ReplaceImage(nameSpace, deployment)
	},
}

func ReplaceImage(nameSpace string, deployment string)  {
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", "/Users/xinxinhuang/.kube/config")
	if err != nil {
		fmt.Printf("%v",err)
		return
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Printf("%v",err)
		return
	}

	//dep, err := clientSet.AppsV1().Deployments(nameSpace).Get(context.TODO(), deployment, metav1.GetOptions{})

	scale, err := clientSet.AppsV1().Deployments(nameSpace).GetScale(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("%v",err)
		return
	}

	fmt.Println("debugging deployment: " + deployment)
	fmt.Sprintf("scaling replicas to 1")
	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now", deployment, scale.Spec.Replicas)
	} else {
		fmt.Printf("deployment %s's replicas is 1\n", deployment)
	}
}