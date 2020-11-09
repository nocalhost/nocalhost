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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/clientgoutils"
	"time"
)

var (
	nameSpace, lang, image string
	mountPath              = "/home/code"
	sidecarImage           = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:v1"
)

func init() {
	devStartCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	devStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&lang, "lang", "l", "", "the development language, eg: java go python")
	devStartCmd.Flags().StringVarP(&image, "image", "i", "", "image of development container")
	devStartCmd.Flags().StringVar(&mountPath, "mount-path", mountPath, "path mounted for sync files")
	devStartCmd.Flags().StringVar(&sidecarImage, "sidecar-image", sidecarImage, "image of sidecar container")
	debugCmd.AddCommand(devStartCmd)
}

var devStartCmd = &cobra.Command{
	Use:   "start",
	Short: "enter dev model",
	Long:  `enter dev model`,
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
		fmt.Println("entering development model...")
		ReplaceImage(nameSpace, deployment)
	},
}

func ReplaceImage(nameSpace string, deployment string) {
	var debugImage string

	switch lang {
	case "java":
		debugImage = "roandocker/share-container-java:v3"
	case "ruby":
		debugImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v1"
	default:
		fmt.Printf("unsupported language : %s\n", lang)
		return
	}

	if image != "" {
		debugImage = image
	}

	clientUtils, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, 0)
	clientgoutils.Must(err)

	deploymentsClient := clientUtils.GetDeploymentClient(nameSpace)

	scale, err := deploymentsClient.GetScale(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	fmt.Println("developing deployment: " + deployment)
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
	dep, err := clientUtils.GetDeployment(context.TODO(), nameSpace, deployment)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", deployment, err)
		return
	}

	volName := "nocalhost-shared-volume"
	// shared volume
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	if dep.Spec.Template.Spec.Volumes == nil {
		debug("volume slice define is nil, init slice")
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, vol)

	// volume mount
	volMount := corev1.VolumeMount{
		Name:      volName,
		MountPath: mountPath,
	}

	// default : replace the first container
	dep.Spec.Template.Spec.Containers[0].Image = debugImage
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, volMount)

	sideCarContainer := corev1.Container{
		Name:    "nocalhost-sidecar",
		Image:   sidecarImage,
		Command: []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"},
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount)
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	//_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	_, err = clientUtils.UpdateDeployment(context.TODO(), nameSpace, dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return
	}

	<-time.NewTimer(time.Second * 3).C

	podList, err := clientUtils.ListPodsOfDeployment(nameSpace, dep.Name)
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
		podList, err = clientUtils.ListPodsOfDeployment(nameSpace, dep.Name)
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

//
//func findPodListOfDeployment(deploy string) ([]v1.Pod, error) {
//	podClient, err := GetPodClient(nameSpace)
//	if err != nil {
//		fmt.Printf("failed to get podList client: %v\n", err)
//		return nil, err
//	}
//
//	podList, err := podClient.List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		fmt.Printf("failed to get pods, err: %v\n", err)
//		return nil, err
//	}
//
//	result := make([]v1.Pod, 0)
//
//OuterLoop:
//	for _, pod := range podList.Items {
//		if pod.OwnerReferences != nil {
//			for _, ref := range pod.OwnerReferences {
//				if ref.Kind == "ReplicaSet" {
//					rss, _ := GetReplicaSetsControlledByDeployment(deploy)
//					if rss != nil {
//						for _, rs := range rss {
//							if rs.Name == ref.Name {
//								result = append(result, pod)
//								continue OuterLoop
//							}
//						}
//					}
//				}
//			}
//		}
//	}
//	return result, nil
//}
