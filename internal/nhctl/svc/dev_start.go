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

package svc

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	putils "nocalhost/pkg/nhctl/utils"
	"strings"
	"time"
)

type DevStartOptions struct {
	WorkDir      string
	SideCarImage string
	DevImage     string
	Container    string
	//SvcType      string

	Kubeconfig string

	// for debug
	SyncthingVersion string

	// Now it's only use to specify the `root dir` user want to sync
	LocalSyncDir  []string
	StorageClass  string
	PriorityClass string
}

// ReplaceImage In DevMode, nhctl will replace the container of your workload with two containers:
// one is called devContainer, the other is called sideCarContainer
func (c *Controller) ReplaceImage(ctx context.Context, ops *DevStartOptions) error {

	var err error
	c.Client.Context(ctx)

	if err = c.markReplicaSetRevision(); err != nil {
		return err
	}

	if err = c.scaleReplicasToOne(ctx); err != nil {
		return err
	}

	devContainer, err := c.findDevContainer(ops.Container)
	if err != nil {
		return err
	}

	devModeVolumes := make([]corev1.Volume, 0)
	devModeMounts := make([]corev1.VolumeMount, 0)

	// Set volumes
	syncthingVolumes, syncthingVolumeMounts := c.generateSyncVolumesAndMounts()
	devModeVolumes = append(devModeVolumes, syncthingVolumes...)
	devModeMounts = append(devModeMounts, syncthingVolumeMounts...)

	workDirAndPersistVolumes, workDirAndPersistVolumeMounts, err := c.genWorkDirAndPVAndMounts(
		ops.Container, ops.StorageClass)

	if err != nil {
		return err
	}
	devModeVolumes = append(devModeVolumes, workDirAndPersistVolumes...)
	devModeMounts = append(devModeMounts, workDirAndPersistVolumeMounts...)

	workDir := c.GetWorkDir(ops.Container)
	devImage := c.GetDevImage(ops.Container) // Default : replace the first container

	sideCarContainer := generateSideCarContainer(workDir)

	devContainer.Image = devImage
	devContainer.Name = "nocalhost-dev"
	devContainer.Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	devContainer.WorkingDir = workDir

	// add env
	devEnv := c.GetDevContainerEnv(ops.Container)
	for _, v := range devEnv.DevEnv {
		env := corev1.EnvVar{Name: v.Name, Value: v.Value}
		devContainer.Env = append(devContainer.Env, env)
	}

	// Add volumeMounts to containers
	devContainer.VolumeMounts = append(devContainer.VolumeMounts, devModeMounts...)
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, devModeMounts...)

	requirements := c.genResourceReq()
	if requirements != nil {
		devContainer.Resources = *requirements
		sideCarContainer.Resources = *requirements
	}

	// todo hxx
	switch c.Type {
	case appmeta.Deployment:
		// Get latest deployment
		dep, err := c.Client.GetDeployment(c.Name)
		if err != nil {
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
			svcProfile, _ := c.GetProfile()
			priorityClass = svcProfile.PriorityClass
		}
		if priorityClass != "" {
			log.Infof("Using priorityClass: %s...", priorityClass)
			dep.Spec.Template.Spec.PriorityClassName = priorityClass
		}

		log.Info("Updating development container...")
		_, err = c.Client.UpdateDeployment(dep, true)
		//specJson, err := json.Marshal(&dep.Spec)
		//if err != nil {
		//	return errors.Wrap(err, "")
		//}
		//err = c.Client.Patch("Deployment", dep.Name, string(specJson))
		if err != nil {
			if strings.Contains(err.Error(), "no PriorityClass") {
				log.Warnf("PriorityClass %s not found, disable it...", priorityClass)
				dep, err = c.Client.GetDeployment(c.Name)
				if err != nil {
					return err
				}
				dep.Spec.Template.Spec.PriorityClassName = ""
				_, err = c.Client.UpdateDeployment(dep, true)
			}
			if err != nil {
				return err
			}
		}
		return c.waitingPodToBeReady()
	default:
		return errors.New(fmt.Sprintf("%s has not support yet", c.Type))

	}
}

func (c *Controller) GetDevContainerEnv(container string) *ContainerDevEnv {
	// Find service env
	devEnv := make([]*profile.Env, 0)
	kvMap := make(map[string]string, 0)
	serviceConfig, _ := c.GetProfile()
	for _, v := range serviceConfig.ContainerConfigs {
		if v.Name == container || container == "" {
			if v.Dev.EnvFrom != nil && len(v.Dev.EnvFrom.EnvFile) > 0 {
				envFiles := make([]string, 0)
				for _, f := range v.Dev.EnvFrom.EnvFile {
					envFiles = append(envFiles, f.Path)
				}
				kvMap = putils.GetKVFromEnvFiles(envFiles)
			}
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

func (c *Controller) markReplicaSetRevision() error {

	// todo hxx
	switch c.Type {
	case appmeta.Deployment:
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
				if _, ok := rs.Annotations[nocalhost.DevImageRevisionAnnotationKey]; ok {
					// already marked
					return nil
				}
			}
			rs := rss[0]
			err = c.Client.Patch("ReplicaSet", rs.Name,
				fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%d", "%s":"%s"}}}`,
					nocalhost.DevImageOriginalPodReplicasAnnotationKey, originalPodReplicas,
					nocalhost.DevImageRevisionAnnotationKey, nocalhost.DevImageRevisionAnnotationValue))
			if err != nil {
				return errors.New("Failed to update rs's annotation :" + err.Error())
			}
			log.Infof("%s has been marked as first revision", rs.Name)
		}
	default:
		return errors.New(fmt.Sprintf("%s has not supported devMode", c.Type))
	}

	return nil
}

func (c *Controller) findDevContainer(containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	// todo hxx
	switch c.Type {
	case appmeta.Deployment:
		dep, err := c.Client.GetDeployment(c.Name)
		if err != nil {
			return nil, err
		}
		if containerName != "" {
			for index, c := range dep.Spec.Template.Spec.Containers {
				if c.Name == containerName {
					return &dep.Spec.Template.Spec.Containers[index], nil
				}
			}
			if devContainer == nil {
				return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
			}
		} else {
			if len(dep.Spec.Template.Spec.Containers) > 1 {
				return nil, errors.New(fmt.Sprintf("There are more than one container defined," +
					"please specify one to start developing"))
			}
			if len(dep.Spec.Template.Spec.Containers) == 0 {
				return nil, errors.New("No container defined ???")
			}
			devContainer = &dep.Spec.Template.Spec.Containers[0]
		}

	default:
		return nil, errors.New(fmt.Sprintf("%s has not support yet", c.Type))

	}

	return devContainer, nil
}

func (c *Controller) scaleReplicasToOne(ctx context.Context) error {

	// todo hxx
	switch c.Type {
	case appmeta.Deployment:
		deploymentsClient := c.Client.GetDeploymentClient()
		scale, err := deploymentsClient.GetScale(ctx, c.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "")
		}

		if scale.Spec.Replicas > 1 {
			log.Infof("Deployment %s's replicas is %d now, scaling it to 1", c.Name, scale.Spec.Replicas)
			scale.Spec.Replicas = 1
			_, err = deploymentsClient.UpdateScale(ctx, c.Name, scale, metav1.UpdateOptions{})
			if err != nil {
				log.Error("Failed to scale replicas to 1")
			} else {
				for i := 0; i < 60; i++ {
					time.Sleep(time.Second * 1)
					scale, err = deploymentsClient.GetScale(ctx, c.Name, metav1.GetOptions{})
					if scale.Spec.Replicas > 1 {
						log.Debugf("Waiting replicas scaling to 1")
					} else {
						log.Info("Replicas has been set to 1")
						break
					}
				}
			}
		} else {
			log.Infof("Deployment %s's replicas is already 1, no need to scale", c.Name)
		}
	default:
		return errors.New(fmt.Sprintf("%s has not supported devMode", c.Type))
	}
	return nil
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

	secretName := ""
	if c.Type == appmeta.Deployment {
		secretName = c.Name + "-" + secret_config.SecretName
	} else {
		secretName = c.Name + "-" + string(c.Type) + "-" + secret_config.SecretName
	}
	defaultMode := int32(nocalhost.DefaultNewFilePermission)
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
			labels[nocalhost.AppLabel] = c.AppName
			labels[nocalhost.ServiceLabel] = c.Name
			labels[nocalhost.PersistentVolumeDirLabel] = utils.Sha1ToString(persistentVolume.Path)
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
				log.Infof("No PVC for %s found, trying to create one...", persistentVolume.Path)
				pvc, err = c.createPvcForPersistentVolumeDir(persistentVolume, labels, storageClass)
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

func (c *Controller) GetPersistentVolumeDirs(container string) []*profile.PersistentVolumeDir {
	svcProfile, _ := c.GetProfile()
	if svcProfile != nil {
		return svcProfile.GetContainerDevConfigOrDefault(container).PersistentVolumeDirs
	}
	return nil
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
	annotations := map[string]string{nocalhost.PersistentVolumeDirLabel: persistentVolume.Path}
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

func generateSideCarContainer(workDir string) corev1.Container {

	sideCarContainer := corev1.Container{
		Name:       "nocalhost-sidecar",
		Image:      nocalhost.DefaultSideCarImage,
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

func (c *Controller) genResourceReq() *corev1.ResourceRequirements {

	var (
		err          error
		requirements *corev1.ResourceRequirements
	)

	svcProfile, _ := c.GetProfile()
	resourceQuota := svcProfile.ContainerConfigs[0].Dev.DevContainerResources

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

func (c *Controller) waitingPodToBeReady() error {
	// Wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start...")
	spinner.Start()

	switch c.Type {
	case appmeta.Deployment:
	wait:
		for {
			<-time.NewTimer(time.Second * 1).C
			// Get the latest revision
			podList, err := c.Client.ListLatestRevisionPodsByDeployment(c.Name)
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
	default:
		return errors.New(fmt.Sprintf("%s has not supported yet", c.Type))
	}
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

func (c *Controller) GetNocalhostDevContainerPod() (string, error) {
	// todo hxx
	var (
		checkPodsList *corev1.PodList
		err           error
	)
	switch c.Type {
	case appmeta.Deployment:
		checkPodsList, err = c.Client.ListPodsByDeployment(c.Name)
	default:
		return "", errors.New("Unsupported type")
	}
	if err != nil {
		return "", err
	}

	found := false
	for _, pod := range checkPodsList.Items {
		if pod.Status.Phase == "Running" {
			for _, container := range pod.Spec.Containers {
				if container.Name == nocalhost.DefaultNocalhostSideCarName {
					found = true
					break
				}
			}
			if found {
				return pod.Name, nil
			}
		}
	}
	return "", errors.New("dev container not found")
}
