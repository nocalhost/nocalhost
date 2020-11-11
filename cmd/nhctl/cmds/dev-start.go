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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"nocalhost/pkg/nhctl/clientgoutils"
	"time"
)

var (
	nameSpace string
)

type DevFlags struct {
	*EnvSettings
	SideCarImage string
	DevImage     string
	DevLang      string
	MountPath    string
	Deployment   string
}

var devFlags = DevFlags{
	EnvSettings: settings,
}

func init() {

	devStartCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	devStartCmd.Flags().StringVarP(&devFlags.Deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&devFlags.DevLang, "lang", "l", "", "the development language, eg: java go python")
	devStartCmd.Flags().StringVarP(&devFlags.DevImage, "image", "i", "", "image of development container")
	devStartCmd.Flags().StringVar(&devFlags.MountPath, "mount-path", devFlags.MountPath, "path mounted for sync files")
	devStartCmd.Flags().StringVar(&devFlags.SideCarImage, "sidecar-image", devFlags.SideCarImage, "image of sidecar container")
	debugCmd.AddCommand(devStartCmd)
}

var devStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "enter dev model",
	Long:  `enter dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		nocalhostConfig = NewNocalHostConfig(GetApplicationNocalhostConfigPath(applicationName))
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		if devFlags.Deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}
		svcConfig := nocalhostConfig.GetSvcConfig(devFlags.Deployment)
		if devFlags.DevLang == "" {
			if svcConfig != nil && svcConfig.DevLang != "" {
				debug("[nocalhost config] reading devLang from config file")
				devFlags.DevLang = svcConfig.DevLang
			} else {
				fmt.Println("[warning] no development language specified")
			}
		}

		if devFlags.DevImage == "" {
			if svcConfig != nil && svcConfig.DevImage != "" {
				devFlags.DevImage = svcConfig.DevImage
			} else if devFlags.DevLang != "" {
				switch devFlags.DevLang {
				case "java":
					devFlags.DevImage = "roandocker/share-container-java:v3"
				case "ruby":
					devFlags.DevImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v1"
				default:
					fmt.Printf("unsupported language : %s\n", devFlags.DevLang)
					return
				}
			} else {
				fmt.Println("[error] you mush specify a devImage by using -i flag or setting devImage in config or specifying a development language")
				return
			}
		}

		if devFlags.SideCarImage == "" {
			if svcConfig != nil && svcConfig.SideCarImage != "" {
				debug("[nocalhost config] reading sideCarImage config")
				devFlags.SideCarImage = svcConfig.SideCarImage
			} else {
				debug("[default config] sideCarImage uses default value")
				devFlags.SideCarImage = DefaultSideCarImage
			}
		}

		if devFlags.MountPath == "" {
			if svcConfig != nil && svcConfig.MountPath != "" {
				debug("[nocalhost config] reading mountPath config")
				devFlags.MountPath = svcConfig.MountPath
			} else {
				debug("[default config] mountPath uses default value")
				devFlags.MountPath = DefalutMountPath
			}
		}

		fmt.Println("entering development model...")
		ReplaceImage(nameSpace, devFlags.Deployment)
	},
}

func ReplaceImage(nameSpace string, deployment string) {

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
		MountPath: devFlags.MountPath,
	}

	// default : replace the first container
	dep.Spec.Template.Spec.Containers[0].Image = devFlags.DevImage
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, volMount)

	debug("disable readiness probes")
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	sideCarContainer := corev1.Container{
		Name:    "nocalhost-sidecar",
		Image:   devFlags.SideCarImage,
		Command: []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"},
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount)
	sideCarContainer.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.IntOrString{
					IntVal: 22,
				},
			},
		},
	}
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
