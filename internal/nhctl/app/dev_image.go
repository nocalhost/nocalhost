/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package app

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"nocalhost/internal/nhctl/profile"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nocalhost/internal/nhctl/coloredoutput"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) markReplicaSetRevision(svcName string) error {

	dep0, err := a.client.GetDeployment(svcName)
	if err != nil {
		return err
	}

	// Mark current revision for rollback
	rss, err := a.client.GetSortedReplicaSetsByDeployment(svcName)
	if err != nil {
		return err
	}
	if len(rss) > 0 {
		// Recording original pod replicas for dev end to recover
		originalPodReplicas := 1
		if dep0.Spec.Replicas != nil {
			originalPodReplicas = int(*dep0.Spec.Replicas)
		}
		firstRevisionName := ""
		for _, rs := range rss {
			if rs.Annotations[DevImageRevisionAnnotationKey] == DevImageRevisionAnnotationValue {
				firstRevisionName = rs.Name
				if rs.Annotations[DevImageOriginalPodReplicasAnnotationKey] == "" {
					rs.Annotations[DevImageOriginalPodReplicasAnnotationKey] = strconv.Itoa(originalPodReplicas)
					_, err = a.client.UpdateReplicaSet(rs)
					if err != nil {
						return errors.New("Failed to update rs's annotation")
					}
				}
				break
			}
		}
		if firstRevisionName != "" {
			log.Debugf("First revision replicaSet %s has already been marked", firstRevisionName)
		} else {
			rs := rss[0]
			rs.Annotations[DevImageRevisionAnnotationKey] = DevImageRevisionAnnotationValue
			rs.Annotations[DevImageOriginalPodReplicasAnnotationKey] = strconv.Itoa(originalPodReplicas)
			// _, err = a.client.ClientSet.AppsV1().
			// ReplicaSets(a.GetNamespace()).Update(ctx, rs, metav1.UpdateOptions{})
			_, err = a.client.UpdateReplicaSet(rs)
			if err != nil {
				return errors.New("Failed to update rs's annotation :" + err.Error())
			} else {
				log.Infof("%s has been marked as first revision", rs.Name)
			}
		}
	}
	return nil
}

// There are two volume used by syncthing in sideCarContainer:
// 1. A EmptyDir volume mounts to /var/syncthing in sideCarContainer
// 2. A volume mounts Secret to /var/syncthing/secret in sideCarContainer
func generateSyncVolumesAndMounts(svcName string) ([]corev1.Volume, []corev1.VolumeMount) {

	syncthingVolumes := make([]corev1.Volume, 0)
	syncthingVolumeMounts := make([]corev1.VolumeMount, 0)

	// syncthing secret volume
	syncthingEmptyDirVol := corev1.Volume{
		Name: secret_config.EmptyDir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	defaultMode := int32(DefaultNewFilePermission)
	syncthingSecretVol := corev1.Volume{
		Name: secret_config.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: svcName + "-" + secret_config.SecretName,
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

	// syncthing volume mount
	syncthingHomeDirVolumeMount := corev1.VolumeMount{
		Name:      syncthingEmptyDirVol.Name,
		MountPath: secret_config.DefaultSyncthingHome,
		SubPath:   "syncthing",
	}

	// syncthing secret volume
	syncthingSecretVolumeMount := corev1.VolumeMount{
		Name:      syncthingSecretVol.Name,
		MountPath: secret_config.DefaultSyncthingSecretHome,
		ReadOnly:  false,
	}

	syncthingVolumes = append(syncthingVolumes, syncthingEmptyDirVol, syncthingSecretVol)
	syncthingVolumeMounts = append(syncthingVolumeMounts, syncthingHomeDirVolumeMount, syncthingSecretVolumeMount)
	return syncthingVolumes, syncthingVolumeMounts
}

// If PVC exists, use it directly
// If PVC not exists, try to create one
// If PVC failed to create, the whole process of entering DevMode will fail
func (a *Application) genWorkDirAndPVAndMounts(svcName, container, storageClass string) (
	[]corev1.Volume, []corev1.VolumeMount, error,
) {

	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)
	workDir := a.GetDefaultWorkDir(svcName, container)

	workDirVol := corev1.Volume{
		Name: "nocalhost-shared-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	var workDirDefinedInPersistVolume bool // if workDir is specified in persistentVolumeDirs
	var workDirResideInPersistVolumeDirs bool
	persistentVolumes := a.GetPersistentVolumeDirs(svcName, container)
	if len(persistentVolumes) > 0 {
		for index, persistentVolume := range persistentVolumes {
			if persistentVolume.Path == "" {
				log.Warnf("PersistentVolume's path should be set")
				continue
			}

			// Check if pvc is already exist
			labels := map[string]string{}
			labels[AppLabel] = a.Name
			labels[ServiceLabel] = svcName
			labels[PersistentVolumeDirLabel] = utils.Sha1ToString(persistentVolume.Path)
			claims, err := a.client.GetPvcByLabels(labels)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("Fail to get a pvc for %s", persistentVolume.Path))
				continue
			}
			if len(claims) > 1 {
				log.Warn(
					fmt.Sprintf(
						"Find %d pvc for %s, expected 1, skipping this dir", len(claims), persistentVolume.Path,
					),
				)
				continue
			}

			var claimName string
			if len(claims) == 1 { // pvc for this path found
				claimName = claims[0].Name
			} else { // no pvc for this path, create one
				var pvc *corev1.PersistentVolumeClaim
				log.Infof("No PVC for %s found, trying to create one...", persistentVolume.Path)
				pvc, err = a.createPvcForPersistentVolumeDir(persistentVolume, labels, storageClass)
				if err != nil || pvc == nil {
					return nil, nil, errors.New("Failed to create pvc for " + persistentVolume.Path)
				}
				claimName = pvc.Name
			}

			// Do not use emptyDir for workDir
			if persistentVolume.Path == workDir {
				workDirDefinedInPersistVolume = true
				workDirVol.EmptyDir = nil
				workDirVol.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				}

				log.Info("WorkDir uses persistent volume defined in persistVolumeDirs")
				continue
			} else if strings.HasPrefix(workDir, persistentVolume.Path) && !workDirDefinedInPersistVolume {
				log.Infof("WorkDir:%s resides in the persist dir: %s", workDir, persistentVolume.Path)
				// No need to mount workDir
				workDirResideInPersistVolumeDirs = true
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

			volumes = append(volumes, persistentVol)
			volumeMounts = append(volumeMounts, persistentMount)

			log.Infof("%s mounts a pvc successfully", persistentVolume.Path)
		}
	}

	if workDirDefinedInPersistVolume || !workDirResideInPersistVolumeDirs {
		if workDirDefinedInPersistVolume {
			log.Info("Mount workDir to persist volume")
		} else {
			log.Info("Mount workDir to emptyDir")
		}
		workDirVolumeMount := corev1.VolumeMount{
			Name:      workDirVol.Name,
			MountPath: workDir,
		}
		volumeMounts = append(volumeMounts, workDirVolumeMount)
		volumes = append(volumes, workDirVol)
	} else {
		log.Debug("No need to mount workDir")
	}

	return volumes, volumeMounts, nil
}

func (a *Application) genResourceReq(svcName string) *corev1.ResourceRequirements {

	var (
		err          error
		requirements *corev1.ResourceRequirements
	)

	svcProfile, _ := a.GetSvcProfile(svcName)
	resourceQuota := svcProfile.ContainerConfigs[0].Dev.DevContainerResources

	if resourceQuota != nil {
		log.Debug("DevContainer uses resource limits defined in config")
		requirements, err = convertResourceQuota(resourceQuota)
		utils.ShouldI(err, "Failed to parse resource requirements")
	}

	return requirements
}

func generateSideCarContainer(workDir string) corev1.Container {

	sideCarContainer := corev1.Container{
		Name:       "nocalhost-sidecar",
		Image:      DefaultSideCarImage,
		WorkingDir: workDir,
	}

	// over write syncthing command
	sideCarContainer.Command = []string{"/bin/sh", "-c"}
	sideCarContainer.Args = []string{
		"unset STGUIADDRESS && cp " + secret_config.DefaultSyncthingSecretHome +
			"/* " + secret_config.DefaultSyncthingHome +
			"/ && /bin/entrypoint.sh && /bin/syncthing -home /var/syncthing",
	}
	return sideCarContainer
}

// In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (a *Application) ReplaceImage(ctx context.Context, svcName string, ops *DevStartOptions) error {

	var err error
	a.client.Context(ctx)

	if err = a.markReplicaSetRevision(svcName); err != nil {
		return err
	}

	if err = a.scaleDeploymentReplicasToOne(ctx, svcName); err != nil {
		return err
	}

	dep, err := a.client.GetDeployment(svcName)
	if err != nil {
		return err
	}

	var devContainer *corev1.Container
	if ops.Container != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				devContainer = &dep.Spec.Template.Spec.Containers[index]
				break
			}
		}
		if devContainer == nil {
			return errors.New(fmt.Sprintf("Container %s not found", ops.Container))
		}
	} else {
		if len(dep.Spec.Template.Spec.Containers) > 1 {
			return errors.New(fmt.Sprintf("There are more than one container defined, please specify one to start developing"))
		}
		if len(dep.Spec.Template.Spec.Containers) == 0 {
			return errors.New("No container defined ???")
		}
		devContainer = &dep.Spec.Template.Spec.Containers[0]
	}

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := generateSyncVolumesAndMounts(svcName)
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := a.genWorkDirAndPVAndMounts(
		svcName, ops.Container, ops.StorageClass,
	)
	if err != nil {
		return err
	}
	devModeVolumes = append(devModeVolumes, workDirAndPersistVolumes...)
	devModeMounts = append(devModeMounts, workDirAndPersistVolumeMounts...)

	workDir := a.GetDefaultWorkDir(svcName, ops.Container)
	devImage := a.GetDefaultDevImage(svcName, ops.Container) // Default : replace the first container

	sideCarContainer := generateSideCarContainer(workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// add env
	devEnv := a.GetDevContainerEnv(svcName, ops.Container)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := a.genResourceReq(svcName)
	if requirements != nil {
		devContainer.Resources = *requirements
		sideCarContainer.Resources = *requirements
	}

	// Get latest deployment
	if dep, err = a.client.GetDeployment(svcName); err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				dep.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		dep.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if dep.Spec.Template.Spec.Volumes == nil {
		log.Debugf("Service %s has no volume", dep.Name)
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	// PriorityClass
	priorityClass := ops.PriorityClass
	if priorityClass == "" {
		svcProfile, _ := a.GetSvcProfile(svcName)
		priorityClass = svcProfile.PriorityClass
	}
	if priorityClass != "" {
		log.Infof("Using priorityClass: %s...", priorityClass)
		dep.Spec.Template.Spec.PriorityClassName = priorityClass
	}

	log.Info("Updating development container...")
	_, err = a.client.UpdateDeployment(dep, true)
	if err != nil {
		if strings.Contains(err.Error(), "no PriorityClass") {
			log.Warnf("PriorityClass %s not found, disable it...", priorityClass)
			dep, err = a.client.GetDeployment(svcName)
			if err != nil {
				return err
			}
			dep.Spec.Template.Spec.PriorityClassName = ""
			_, err = a.client.UpdateDeployment(dep, true)
		}
		if err != nil {
			return err
		}
	}
	return a.waitingPodOfDeploymentToBeReady(dep.Name)
}

func (a *Application) waitingPodOfDeploymentToBeReady(deployName string) error {
	// Wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start...")
	spinner.Start()

wait:
	for {
		<-time.NewTimer(time.Second * 1).C
		// Get the latest revision
		podList, err := a.client.ListLatestRevisionPodsByDeployment(deployName)
		if err != nil {
			return err
		}
		if len(podList) == 1 {
			pod := podList[0]
			if pod.Status.Phase != corev1.PodRunning {
				spinner.Update(fmt.Sprintf("Waiting for pod %s to be running", pod.Name))
				continue
			}
			if len(pod.Spec.Containers) == 0 {
				return errors.New(fmt.Sprintf("%s has no container ???", pod.Name))
			}

			// Make sure all containers are ready and running
			for _, c := range pod.Spec.Containers {
				if !isContainerReadyAndRunning(c.Name, &pod) {
					spinner.Update(fmt.Sprintf("Container %s is not ready, waiting...", c.Name))
					continue wait
				}
			}
			spinner.Update("All containers are ready")
			break
		} else {
			spinner.Update(fmt.Sprintf("Waiting pod to be replaced..."))
		}
	}
	spinner.Stop()
	coloredoutput.Success("Development container has been updated")
	return nil
}

func convertResourceQuota(quota *profile.ResourceQuota) (*corev1.ResourceRequirements, error) {
	var err error
	requirements := &corev1.ResourceRequirements{}

	if quota.Requests != nil {
		requirements.Requests, err = convertToResourceList(quota.Requests.Cpu, quota.Requests.Memory)
		if err != nil {
			return nil, err
		}
	}

	if quota.Limits != nil {
		requirements.Limits, err = convertToResourceList(quota.Limits.Cpu, quota.Limits.Memory)
		if err != nil {
			return nil, err
		}
	}
	if len(requirements.Limits) == 0 && len(requirements.Requests) == 0 {
		return nil, errors.New("Resource requirements not defined")
	}
	return requirements, nil
}

func convertToResourceList(cpu string, mem string) (corev1.ResourceList, error) {
	requestMap := make(map[corev1.ResourceName]resource.Quantity, 0)
	if mem != "" {
		q, err := resource.ParseQuantity(mem)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		requestMap[corev1.ResourceMemory] = q
	}
	if cpu != "" {
		q, err := resource.ParseQuantity(cpu)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		requestMap[corev1.ResourceCPU] = q
	}
	return requestMap, nil
}

// Initial a pvc for persistent volume dir, and waiting util pvc succeed to bound to a pv
// If pvc failed to bound to a pv, the pvc will been deleted, and return nil
func (a *Application) createPvcForPersistentVolumeDir(
	persistentVolume *profile.PersistentVolumeDir, labels map[string]string, storageClass string,
) (*corev1.PersistentVolumeClaim, error) {
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
		pvc, err = a.client.CreatePVC(pvcName, labels, annotations, capacity, nil)
	} else {
		pvc, err = a.client.CreatePVC(pvcName, labels, annotations, capacity, &storageClass)
	}
	if err != nil {
		return nil, err
	}
	// wait pvc to be ready
	var pvcBounded bool
	var errorMes string

	for i := 0; i < 30; i++ {
		time.Sleep(time.Second * 2)
		pvc, err = a.client.GetPvcByName(pvc.Name)
		if err != nil {
			log.Warnf("Failed to update pvc's status: %s", err.Error())
			continue
		}
		if pvc.Status.Phase == corev1.ClaimBound {
			log.Infof("PVC %s has bounded to a pv", pvc.Name)
			pvcBounded = true
			return pvc, nil
		} else {
			if len(pvc.Status.Conditions) > 0 {
				for _, condition := range pvc.Status.Conditions {
					errorMes = condition.Message
					if condition.Reason == "ProvisioningFailed" {
						log.Warnf(
							"Failed to create a pvc for %s, check if your StorageClass is set correctly",
							persistentVolume.Path,
						)
						break
					}
				}
			}
			log.Infof("PVC %s's status is %s, waiting it to be bounded", pvc.Name, pvc.Status.Phase)
		}
	}
	if !pvcBounded {
		if errorMes == "" {
			errorMes = "timeout"
		}
		log.Warnf("Failed to wait %s to be bounded: %s", pvc.Name, errorMes)
		err = a.client.DeletePVC(pvc.Name)
		if err != nil {
			log.Warnf("Fail to clean pvc %s", pvc.Name)
			return nil, err
		} else {
			log.Infof("PVC %s is cleaned up", pvc.Name)
		}
	}
	return nil, errors.New("Failed to create pvc for " + persistentVolume.Path)
}

func (a *Application) scaleDeploymentReplicasToOne(ctx context.Context, deployment string) error {

	deploymentsClient := a.client.GetDeploymentClient()
	scale, err := deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "")
	}

	if scale.Spec.Replicas > 1 {
		log.Infof("Deployment %s's replicas is %d now, scaling it to 1", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(ctx, deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			log.Error("Failed to scale replicas to 1")
		} else {
			for i := 0; i < 60; i++ {
				time.Sleep(time.Second * 1)
				scale, err = deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
				if scale.Spec.Replicas > 1 {
					log.Debugf("Waiting replicas scaling to 1")
				} else {
					log.Info("Replicas has been set to 1")
					break
				}
			}
		}
	} else {
		log.Infof("Deployment %s's replicas is already 1, no need to scale", deployment)
	}
	return nil
}
