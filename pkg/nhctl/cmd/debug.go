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
var nameSpace, lang, image string

func init() {
	//install k8s
	debugCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	debugCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	debugCmd.Flags().StringVarP(&lang, "type", "l", "", "the development language, eg: java go python")
	debugCmd.Flags().StringVarP(&image, "image", "i", "", "image of development container")

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
		if deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}
		if lang == "" {
			fmt.Println("error: please use -l to specify your development language")
			return
		}
		fmt.Println("enter debug...")
		ReplaceImage(nameSpace, deployment)
	},
}

func ReplaceImage(nameSpace string, deployment string)  {
	var debugImage string

	switch lang {
	case "java":
		debugImage = "roandocker/share-container-java:v2"
	default:
		fmt.Printf("unsupported language : %s\n", lang)
		return
	}

	if image != "" {
		debugImage = image
	}

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
	fmt.Println("scaling replicas to 1")
	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = clientSet.AppsV1().Deployments(nameSpace).UpdateScale(context.TODO(), deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		}else {
			fmt.Println("replicas has been scaled to 1")
		}
	} else {
		fmt.Printf("deployment %s's replicas is already 1\n", deployment)
	}

	fmt.Println("Update develop container...")
	dep, err := clientSet.AppsV1().Deployments(nameSpace).Get(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get deployment %d , err : %v\n", deployment, err)
	}

	// default : replace the first container
	//container := dep.Spec.Template.Spec.Containers[0]
	dep.Spec.Template.Spec.Containers[0].Image = debugImage
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c",  "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"}

	_, err = clientSet.AppsV1().Deployments(nameSpace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
	}else {
		fmt.Println("develop container has been updated")
	}

}