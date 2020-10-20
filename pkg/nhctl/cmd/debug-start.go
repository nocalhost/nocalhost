package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var nameSpace, lang, image string

func init() {
	debugStartCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	debugStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	debugStartCmd.Flags().StringVarP(&lang, "type", "l", "", "the development language, eg: java go python")
	debugStartCmd.Flags().StringVarP(&image, "image", "i", "", "image of development container")
	debugStartCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	debugCmd.AddCommand(debugStartCmd)
}

var debugStartCmd = &cobra.Command{
	Use:   "start",
	Short: "enter debug model",
	Long:  `enter debug model`,
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

func ReplaceImage(nameSpace string, deployment string) {
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

	deploymentsClient, err := GetDeploymentClient(nameSpace)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	scale, err := deploymentsClient.GetScale(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	fmt.Println("debugging deployment: " + deployment)
	fmt.Println("scaling replicas to 1")
	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(context.TODO(), deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		} else {
			time.Sleep(time.Second * 5)
			fmt.Println("replicas has been scaled to 1")
		}
	} else {
		fmt.Printf("deployment %s's replicas is already 1\n", deployment)
	}

	fmt.Println("Updating develop container...")
	dep, err := deploymentsClient.Get(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", deployment, err)
	}

	// default : replace the first container
	dep.Spec.Template.Spec.Containers[0].Image = debugImage
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"}

	_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return
	}

	<-time.NewTimer(time.Second * 3).C

	podList, err := findPodListOfDeployment(dep.Name)
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return
	}

	fmt.Printf("%d pod found\n", len(podList)) // should be 2

	//pod := podList.Items[0]
	// wait podList to be ready
	fmt.Printf("waiting pod to start.")
	for {
		<-time.NewTimer(time.Second * 2).C
		podList, err = findPodListOfDeployment(dep.Name)
		if err != nil {
			fmt.Printf("failed to get pods, err: %v\n", err)
			return
		}
		if len(podList) == 1 {
			// todo check container status
			break
		}
		fmt.Print(".")
	}

	fmt.Println("develop container has been updated")
}

func findPodListOfDeployment(deploy string) ([]v1.Pod, error) {
	podClient, err := GetPodClient(nameSpace)
	if err != nil {
		fmt.Printf("failed to get podList client: %v\n", err)
		return nil, err
	}

	podList, err := podClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return nil, err
	}

	result := make([]v1.Pod, 0)

	OuterLoop:
	for _, pod := range podList.Items {
		if pod.OwnerReferences != nil {
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "ReplicaSet" {
					rss, _ := GetReplicaSetsControlledByDeployment(deploy)
					if rss != nil {
						for _, rs := range rss {
							if rs.Name == ref.Name {
								result = append(result, pod)
								continue OuterLoop
							}
						}
					}
				}
			}
		}
	}
	return result,nil
}
