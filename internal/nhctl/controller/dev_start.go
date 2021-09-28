/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"nocalhost/internal/nhctl/const"

	//"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"time"
)

func (c *Controller) GetDevContainerEnv(container string) *ContainerDevEnv {
	// Find service env
	devEnv := make([]*profile.Env, 0)
	kvMap := make(map[string]string, 0)
	//serviceConfig, _ := c.GetProfile()
	serviceConfig, _ := c.GetConfig()
	for _, v := range serviceConfig.ContainerConfigs {
		if v.Name == container || container == "" {
			// Env has a higher priority than envFrom
			for _, env := range v.Dev.Env {
				kvMap[env.Name] = env.Value
			}
		}
	}
	for k, v := range kvMap {
		env := &profile.Env{
			Name:  k,
			Value: v,
		}
		devEnv = append(devEnv, env)
	}
	return &ContainerDevEnv{DevEnv: devEnv}
}

func (c *Controller) GetDevSidecarImage(container string) string {
	// Find service env
	serviceConfig, _ := c.GetConfig()
	for _, v := range serviceConfig.ContainerConfigs {
		if v.Name == container || container == "" {
			// Env has a higher priority than envFrom
			if v.Dev != nil {
				return v.Dev.SidecarImage
			}
		}
	}

	return ""
}

func (c *Controller) markReplicaSetRevision() error {

	dep0, err := c.Client.GetDeployment(c.Name)
	if err != nil {
		return err
	}

	// Mark current revision for rollback
	rss, err := c.Client.GetSortedReplicaSetsByDeployment(c.Name)
	if err != nil {
		return err
	}
	if len(rss) > 0 {
		// Recording original pod replicas for dev end to recover
		originalPodReplicas := 1
		if dep0.Spec.Replicas != nil {
			originalPodReplicas = int(*dep0.Spec.Replicas)
		}
		for _, rs := range rss {
			if _, ok := rs.Annotations[_const.DevImageRevisionAnnotationKey]; ok {
				// already marked
				return nil
			}
		}
		rs := rss[0]
		retryTimes := 5
		for i := 0; i < retryTimes; i++ {
			time.Sleep(time.Second * 1)
			if err = c.Client.Patch(
				"ReplicaSet", rs.Name,
				fmt.Sprintf(
					`{"metadata":{"annotations":{"%s":"%d", "%s":"%s"}}}`,
					_const.DevImageOriginalPodReplicasAnnotationKey, originalPodReplicas,
					_const.DevImageRevisionAnnotationKey, _const.DevImageRevisionAnnotationValue,
				),
			); err == nil {
				break
			}
		}
		if err != nil {
			return errors.New("Failed to update rs's annotation :" + err.Error())
		}
		log.Infof("%s has been marked as first revision", rs.Name)
	}
	return nil
}

func (c *Controller) GetSyncThingSecretName() string {
	return c.Name + "-" + c.Type.String() + "-" + secret_config.SecretName
}

// There are two volume used by syncthing in sideCarContainer:
// 1. A EmptyDir volume mounts to /var/syncthing in sideCarContainer
// 2. A volume mounts Secret to /var/syncthing/secret in sideCarContainer
func (c *Controller) generateSyncVolumesAndMounts() ([]corev1.Volume, []corev1.VolumeMount) {

	syncthingVolumes := make([]corev1.Volume, 0)
	syncthingVolumeMounts := make([]corev1.VolumeMount, 0)

	// syncthing secret volume
	syncthingEmptyDirVol := corev1.Volume{
		Name: secret_config.EmptyDir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	secretName := c.GetSyncThingSecretName()
	defaultMode := int32(_const.DefaultNewFilePermission)
	syncthingSecretVol := corev1.Volume{
		Name: secret_config.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
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
func (c *Controller) genWorkDirAndPVAndMounts(container, storageClass string) (
	[]corev1.Volume, []corev1.VolumeMount, error) {

	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)
	workDir := c.GetWorkDir(container)

	workDirVol := corev1.Volume{
		Name: "nocalhost-shared-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	var workDirDefinedInPersistVolume bool // if workDir is specified in persistentVolumeDirs
	var workDirResideInPersistVolumeDirs bool
	persistentVolumes := c.GetPersistentVolumeDirs(container)
	if len(persistentVolumes) > 0 {
		for index, persistentVolume := range persistentVolumes {
			if persistentVolume.Path == "" {
				log.Warnf("PersistentVolume's path should be set")
				continue
			}

			// Check if pvc is already exist
			labels := map[string]string{}
			labels[_const.AppLabel] = c.AppName
			labels[_const.ServiceLabel] = c.Name
			labels[_const.PersistentVolumeDirLabel] = utils.Sha1ToString(persistentVolume.Path)
			claims, err := c.Client.GetPvcByLabels(labels)
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
				if c.GetStorageClass(container) != "" {
					storageClass = c.GetStorageClass(container)
				}
				log.Infof(
					"No PVC for %s found, trying to create one with storage class %s...",
					persistentVolume.Path, storageClass,
				)
				pvc, err = c.createPvcForPersistentVolumeDir(persistentVolume, labels, storageClass)
				if err != nil || pvc == nil {
					return nil, nil, errors.Wrap(
						nocalhost.CreatePvcFailed, "Failed to create pvc for "+persistentVolume.Path,
					)
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

// Initial a pvc for persistent volume dir, and waiting util pvc succeed to bound to a pv
// If pvc failed to bound to a pv, the pvc will been deleted, and return nil
func (c *Controller) createPvcForPersistentVolumeDir(
	persistentVolume *profile.PersistentVolumeDir, labels map[string]string, storageClass string,
) (*corev1.PersistentVolumeClaim, error) {
	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)

	pvcName := fmt.Sprintf("%s-%d", c.AppName, time.Now().UnixNano())
	annotations := map[string]string{_const.PersistentVolumeDirLabel: persistentVolume.Path}
	capacity := persistentVolume.Capacity
	if persistentVolume.Capacity == "" {
		capacity = "10Gi"
	}

	if storageClass == "" {
		pvc, err = c.Client.CreatePVC(pvcName, labels, annotations, capacity, nil)
	} else {
		pvc, err = c.Client.CreatePVC(pvcName, labels, annotations, capacity, &storageClass)
	}
	if err != nil {
		return nil, err
	}
	// wait pvc to be ready
	var pvcBounded bool
	var errorMes string

	for i := 0; i < 30; i++ {
		time.Sleep(time.Second * 2)
		pvc, err = c.Client.GetPvcByName(pvc.Name)
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
		err = c.Client.DeletePVC(pvc.Name)
		if err != nil {
			log.Warnf("Fail to clean pvc %s", pvc.Name)
			return nil, err
		} else {
			log.Infof("PVC %s is cleaned up", pvc.Name)
		}
	}
	return nil, errors.New("Failed to create pvc for " + persistentVolume.Path)
}

func generateSideCarContainer(sidecarImage, workDir string) corev1.Container {
	if sidecarImage == "" {
		sidecarImage = _const.DefaultSideCarImage
	}

	sideCarContainer := corev1.Container{
		Name:       "nocalhost-sidecar",
		Image:      sidecarImage,
		WorkingDir: workDir,
	}

	// over write syncthing command
	sideCarContainer.Command = []string{"/bin/sh", "-c"}
	sideCarContainer.Args = []string{
		"rc-service sshd restart && unset STGUIADDRESS && cp " + secret_config.DefaultSyncthingSecretHome +
			"/* " + secret_config.DefaultSyncthingHome +
			"/ && /bin/entrypoint.sh && /bin/syncthing -home /var/syncthing",
	}
	return sideCarContainer
}

func (c *Controller) genResourceReq(container string) *corev1.ResourceRequirements {

	var (
		err          error
		requirements *corev1.ResourceRequirements
	)

	svcProfile, _ := c.GetConfig()
	if svcProfile == nil {
		return requirements
	}

	containerConfig := svcProfile.GetContainerDevConfigOrDefault(container)
	if containerConfig == nil {
		return requirements
	}

	resourceQuota := containerConfig.DevContainerResources
	if resourceQuota != nil {
		log.Debug("DevContainer uses resource limits defined in config")
		requirements, err = convertResourceQuota(resourceQuota)
		utils.ShouldI(err, "Failed to parse resource requirements")
	}

	return requirements
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

func waitingPodToBeReady(f func() (string, error)) error {
	// Wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start...")
	spinner.Start()

	for i := 0; i < 300; i++ {
		<-time.NewTimer(time.Second * 1).C
		if _, err := f(); err == nil {
			break
		}
	}
	spinner.Stop()
	return nil
}

func isContainerReadyAndRunning(containerName string, pod *corev1.Pod) bool {
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName && status.Ready && status.State.Running != nil {
			return true
		}
	}
	return false
}

func findDevPod(podList []corev1.Pod) (string, error) {
	for _, pod := range podList {
		if pod.Status.Phase == "Running" && pod.DeletionTimestamp == nil {
			for _, container := range pod.Spec.Containers {
				if container.Name == _const.DefaultNocalhostSideCarName {
					return pod.Name, nil
				}
			}
		}
	}
	return "", errors.New("dev container not found")
}

func (c *Controller) genContainersAndVolumes(devContainer *corev1.Container,
	containerName, devImage, storageClass string) (*corev1.Container, *corev1.Container, []corev1.Volume, error) {

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := c.generateSyncVolumesAndMounts()
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := c.genWorkDirAndPVAndMounts(
		containerName, storageClass,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	devModeVolumes = append(devModeVolumes, workDirAndPersistVolumes...)
	devModeMounts = append(devModeMounts, workDirAndPersistVolumeMounts...)

	workDir := c.GetWorkDir(containerName)

	if devImage == "" {
		devImage = c.GetDevImage(containerName) // Default : replace the first container
	}

	sideCarContainer := generateSideCarContainer(c.GetDevSidecarImage(containerName), workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// set image pull policy
	sideCarContainer.ImagePullPolicy = _const.DefaultSidecarImagePullPolicy
	devContainer.ImagePullPolicy = _const.DefaultSidecarImagePullPolicy

	// add env
	devEnv := c.GetDevContainerEnv(containerName)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := c.genResourceReq(containerName)
	if requirements != nil {
		devContainer.Resources = *requirements
	}

	if IsResourcesLimitTooLow(&devContainer.Resources) {
		limits := ""
		if devContainer.Resources.Limits != nil {
			if devContainer.Resources.Limits.Cpu() != nil {
				limits += devContainer.Resources.Limits.Cpu().String() + " cpu"
			}
			if devContainer.Resources.Limits.Memory() != nil && devContainer.Resources.Limits.Memory().String() != "0" {
				if len(limits) > 0 {
					limits += ", "
				}
				limits += devContainer.Resources.Limits.Memory().String() + " memory"
			}
		}
		log.PWarnf(
			`Resources Limits: %s is less than the recommended minimum: 2 cpu, 2Gi memory. `+
				"Running programs in DevContainer may fail. You can increase Resource Limits in Nocalhost Config",
			limits,
		)
	}

	r := &profile.ResourceQuota{
		Limits:   &profile.QuotaList{Memory: "1Gi", Cpu: "1"},
		Requests: &profile.QuotaList{Memory: "50Mi", Cpu: "100m"},
	}
	rq, _ := convertResourceQuota(r)
	sideCarContainer.Resources = *rq
	return devContainer, &sideCarContainer, devModeVolumes, nil
}

// IsResourcesLimitTooLow
// Check if resource limit is lower than 2 cpu, 2Gi men
func IsResourcesLimitTooLow(r *corev1.ResourceRequirements) bool {
	if r == nil || r.Limits == nil {
		return false
	}
	if r.Limits.Memory() != nil {
		q, _ := resource.ParseQuantity("2Gi")
		if r.Limits.Memory().Cmp(q) < 0 {
			return true
		}
	}
	if r.Limits.Cpu() != nil {
		q, _ := resource.ParseQuantity("2")
		if r.Limits.Cpu().Cmp(q) < 0 {
			return true
		}
	}
	return false
}
