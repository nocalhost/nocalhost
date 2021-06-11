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

package controller

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
)

type CronJobController struct {
	*Controller
}

const cronjobGeneratedJobPrefix = "cronjob-generated-job-"

func (j *CronJobController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList)
}

func (j *CronJobController) getGeneratedJobName() string {
	return fmt.Sprintf("%s%s", cronjobGeneratedJobPrefix, j.Name())
}

// ReplaceImage For Job, we can't replace the Job' image
// but create a job with dev container instead
func (j *CronJobController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	j.Client.Context(ctx)
	originJob, err := j.Client.GetCronJobs(j.Name())
	if err != nil {
		return err
	}

	originJob.Spec.Schedule = ""

	// Create a Job from origin Job's spec
	generatedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   j.getGeneratedJobName(),
			Labels: map[string]string{nocalhost.DevWorkloadIgnored: "true"},
		},
		Spec: batchv1.JobSpec{
			Selector: originJob.Spec.JobTemplate.Spec.Selector,
			Template: originJob.Spec.JobTemplate.Spec.Template,
		},
	}

	delete(generatedJob.Spec.Template.ObjectMeta.Labels, "controller-uid")
	delete(generatedJob.Spec.Template.ObjectMeta.Labels, "job-name")
	delete(generatedJob.Spec.Selector.MatchLabels, "controller-uid")

	devContainer, err := findContainerInJobSpec(generatedJob, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		j.genContainersAndVolumes(devContainer, ops.Container, ops.StorageClass)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range generatedJob.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				generatedJob.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		generatedJob.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if generatedJob.Spec.Template.Spec.Volumes == nil {
		generatedJob.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	generatedJob.Spec.Template.Spec.Volumes = append(generatedJob.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	generatedJob.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(generatedJob.Spec.Template.Spec.Containers); i++ {
		generatedJob.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		generatedJob.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		generatedJob.Spec.Template.Spec.Containers[i].StartupProbe = nil
		generatedJob.Spec.Template.Spec.Containers[i].SecurityContext = nil
	}

	generatedJob.Spec.Template.Spec.Containers =
		append(generatedJob.Spec.Template.Spec.Containers, *sideCarContainer)

	// Create generated deployment
	if _, err = j.Client.CreateJob(generatedJob); err != nil {
		return err
	}

	return waitingPodToBeReady(j.GetNocalhostDevContainerPod)
}

func (j *CronJobController) Name() string {
	return j.Controller.Name
}

func (j *CronJobController) RollBack(reset bool) error {
	return j.Client.DeleteJob(j.getGeneratedJobName())
}

// GetPodList
// In DevMode, return pod list of generated Job.
// Otherwise, return pod list of original Job
func (j *CronJobController) GetPodList() ([]corev1.Pod, error) {
	if j.IsInDevMode() {
		pl, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
		if err != nil {
			return nil, err
		}
		return pl.Items, nil
	}
	pl, err := j.Client.ListPodsByCronJob(j.Name())
	if err != nil {
		return nil, err
	}
	return pl.Items, nil
}
