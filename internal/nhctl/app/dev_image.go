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

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nocalhost/internal/nhctl/coloredoutput"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) ReplaceImage(ctx context.Context, deployment string, ops *DevStartOptions) error {

	// mark current revision for rollback
	rss, err := a.client.GetSortedReplicaSetsByDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		return err
	}
	if rss != nil && len(rss) > 0 {
		rs := rss[len(rss)-1]
		rs.Annotations[DevImageFlagAnnotationKey] = DevImageFlagAnnotationValue
		_, err = a.client.ClientSet.AppsV1().ReplicaSets(a.GetNamespace()).Update(ctx, rs, metav1.UpdateOptions{})
		if err != nil {
			return errors.New("fail to update rs's annotation")
		}
	}

	err = a.scaleDeploymentReplicasToOne(ctx, deployment)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	log.Info("Updating development container...")
	dep, err := a.client.GetDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		return err
	}

	volName := "nocalhost-shared-volume"
	// shared volume
	workDirVol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	// syncthing secret volume
	syncthingDir := corev1.Volume{
		Name: secret_config.EmptyDir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	defaultMode := int32(DefaultNewFilePermission)
	syncthingVol := corev1.Volume{
		Name: secret_config.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: deployment + "-" + secret_config.SecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  "config.xml",
						Path: "config.xml",
						Mode: &defaultMode,
					},
					{
						Key:  "cert.pem",
						Path: "cert.pem",
						Mode: &defaultMode,
					},
					{
						Key:  "key.pem",
						Path: "key.pem",
						Mode: &defaultMode,
					},
				},
				DefaultMode: &defaultMode,
			},
		},
	}

	if dep.Spec.Template.Spec.Volumes == nil {
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, workDirVol, syncthingVol, syncthingDir)

	// syncthing volume mount
	syncthingVolHomeDirMount := corev1.VolumeMount{
		Name:      secret_config.EmptyDir,
		MountPath: secret_config.DefaultSyncthingHome,
		SubPath:   "syncthing",
	}

	// syncthing secret volume
	syncthingVolMount := corev1.VolumeMount{
		Name:      secret_config.SecretName,
		MountPath: secret_config.DefaultSyncthingSecretHome,
		ReadOnly:  false,
	}

	// volume mount
	workDir := a.GetDefaultWorkDir(deployment)
	if ops.WorkDir != "" {
		workDir = ops.WorkDir
	}

	volMount := corev1.VolumeMount{
		Name:      volName,
		MountPath: workDir,
	}

	// default : replace the first container
	devImage := a.GetDefaultDevImage(deployment)
	if ops.DevImage != "" {
		devImage = ops.DevImage
	}

	devContainer := &dep.Spec.Template.Spec.Containers[0]
	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, volMount)
	// delete users SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// set the entry
	dep.Spec.Template.Spec.Containers[0].WorkingDir = workDir

	// disable readiness probes
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	sideCarImage := a.GetDefaultSideCarImage(deployment)
	if ops.SideCarImage != "" {
		sideCarImage = ops.SideCarImage
	}
	sideCarContainer := corev1.Container{
		Name:       "nocalhost-sidecar",
		Image:      sideCarImage,
		WorkingDir: workDir,
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount, syncthingVolMount, syncthingVolHomeDirMount)

	// add persistent volumes
	persistentVolumes := a.GetPersistentVolumeDirs(deployment)
	if len(persistentVolumes) > 0 {
		for index, persistentVolume := range persistentVolumes {
			if persistentVolume.Path == "" {
				log.Warnf("persistentVolume's path should be set")
				continue
			}

			// check if pvc is already exist
			labels := map[string]string{}
			labels[AppLabel] = a.Name
			labels[ServiceLabel] = deployment
			labels[PersistentVolumeDirLabel] = utils.Sha1ToString(persistentVolume.Path)
			claims, err := a.client.GetPvcByLabels(ctx, a.GetNamespace(), labels)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("fail to get a pvc for %s", persistentVolume.Path))
				continue
			}
			if len(claims) > 1 {
				log.Warn(fmt.Sprintf("find %d pvc for %s, expected 1, skipping this dir", len(claims), persistentVolume.Path))
				continue
			}

			var claimName string
			if len(claims) == 1 { // pvc for this path found
				claimName = claims[0].Name
			} else { // no pvc for this path, create one
				var pvc *corev1.PersistentVolumeClaim
				log.Infof("No PVC for %s found, trying to create one...", persistentVolume.Path)
				pvc, err = a.createPvcForPersistentVolumeDir(ctx, persistentVolume, labels, ops.StorageClass)
				if err != nil || pvc == nil {
					continue
				}
				claimName = pvc.Name
			}

			if persistentVolume.Path == workDir {
				workDirVol.VolumeSource = corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					},
				}
				log.Debug("WorkDir uses persistent volume")
				continue
			}

			persistVolName := fmt.Sprintf("persist-volume-%d", index)
			persistentVol := corev1.Volume{
				Name: persistVolName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					},
				},
			}
			persistentMount := corev1.VolumeMount{
				Name:      persistVolName,
				MountPath: persistentVolume.Path,
			}

			// add volume to pod
			dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, persistentVol)

			// add volume mount to dev container and sidecar container
			devContainer.VolumeMounts = append(devContainer.VolumeMounts, persistentMount)
			sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, persistentMount)

			log.Debugf("%s mount a pvc successfully", persistentVolume.Path)
		}
	}

	// over write syncthing command
	sideCarContainer.Command = []string{"/bin/sh", "-c"}
	sideCarContainer.Args = []string{"unset STGUIADDRESS && cp " + secret_config.DefaultSyncthingSecretHome + "/* " + secret_config.DefaultSyncthingHome + "/ && /bin/entrypoint.sh && /bin/syncthing -home /var/syncthing"}
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	_, err = a.client.UpdateDeployment(ctx, a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return err
	}

	podList, err := a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return err
	}

	log.Debugf("%d pod found", len(podList)) // should be 2

	// wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start...")
	spinner.Start()

wait:
	for {
		podList, err = a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
		if err != nil {
			fmt.Printf("failed to get pods, err: %v\n", err)
			return err
		}
		if len(podList) == 1 {
			pod := podList[0]
			if pod.Status.Phase != corev1.PodRunning {
				spinner.Update(fmt.Sprintf("waiting for pod %s to be Running", pod.Name))
				continue
			}
			if len(pod.Spec.Containers) == 0 {
				log.Fatalf("%s has no container ???", pod.Name)
			}

			// make sure all containers are ready and running
			for _, c := range pod.Spec.Containers {
				if !isContainerReadyAndRunning(c.Name, &pod) {
					spinner.Update(fmt.Sprintf("container %s is not ready, waiting...", c.Name))
					break wait
				}
			}
			spinner.Update("all containers are ready")
			break
		} else {
			spinner.Update(fmt.Sprintf("waiting pod to be replaced..."))
		}
		<-time.NewTimer(time.Second * 1).C
	}
	spinner.Stop()
	coloredoutput.Success("development container has been updated")
	return nil
}

// Create a pvc for persistent volume dir, and waiting util pvc succeed to bound to a pv
// If pvc failed to bound to a pv, the pvc will been deleted, and return nil
func (a *Application) createPvcForPersistentVolumeDir(ctx context.Context, persistentVolume *PersistentVolumeDir, labels map[string]string, storageClass string) (*corev1.PersistentVolumeClaim, error) {
	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)

	pvcName := fmt.Sprintf("%s-%d", a.Name, time.Now().UnixNano())
	annotations := map[string]string{PersistentVolumeDirLabel: persistentVolume.Path}
	capacity := persistentVolume.Capacity
	if persistentVolume.Capacity == "" {
		capacity = "10Gi"
	}

	if storageClass == "" {
		pvc, err = a.client.CreatePVC(a.GetNamespace(), pvcName, labels, annotations, capacity, nil)
	} else {
		pvc, err = a.client.CreatePVC(a.GetNamespace(), pvcName, labels, annotations, capacity, &storageClass)
	}
	if err != nil {
		return nil, err
	}
	// wait pvc to be ready
	var pvcBounded bool
	var errorMes string

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 2)
		pvc, err = a.client.GetPvcByName(ctx, a.GetNamespace(), pvc.Name)
		if err != nil {
			log.Warnf("fail to update pvc's status: %s", err.Error())
			continue
		}
		if pvc.Status.Phase == corev1.ClaimBound {
			log.Infof("pvc %s has bounded to a pv", pvc.Name)
			pvcBounded = true
			//break
			return pvc, nil
		} else {
			if len(pvc.Status.Conditions) > 0 {
				for _, condition := range pvc.Status.Conditions {
					errorMes = condition.Message
					if condition.Reason == "ProvisioningFailed" {
						log.Warnf("fail to create a pvc for %s, check if your StorageClass is set correctly", persistentVolume.Path)
						break
					}
				}
			}
			log.Infof("pvc %s's status is %s, waiting it to be bounded", pvc.Name, pvc.Status.Phase)
		}
	}
	if !pvcBounded {
		if errorMes == "" {
			errorMes = "timeout"
		}
		log.Warnf("fail to wait %s to be bounded: %s", pvc.Name, errorMes)
		err = a.client.DeletePVC(a.GetNamespace(), pvc.Name)
		if err != nil {
			log.Warnf("fail to clean pvc %s", pvc.Name)
		} else {
			log.Infof("pvc %s clean up", pvc.Name)
		}
	}
	return nil, nil
}

func (a *Application) scaleDeploymentReplicasToOne(ctx context.Context, deployment string) error {

	deploymentsClient := a.client.GetDeploymentClient(a.GetNamespace())
	scale, err := deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	log.Info("Scaling replicas to 1")

	if scale.Spec.Replicas > 1 {
		log.Infof("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(ctx, deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		} else {
			//time.Sleep(time.Second * 5) // todo check replicas
			for i := 0; i < 60; i++ {
				time.Sleep(time.Second * 2)
				scale, err = deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
				if scale.Spec.Replicas > 1 {
					log.Debugf("Waiting replicas scaling to 1")
				} else {
					log.Info("replicas has been set to 1")
					break
				}
			}
		}
	} else {
		log.Infof("deployment %s's replicas is already 1\n", deployment)
	}
	return nil
}
