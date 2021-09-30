/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
)

type JobController struct {
	*Controller
}

const jobGeneratedJobPrefix = "job-generated-job-"

func (j *JobController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList.Items)
}

func (j *JobController) getGeneratedJobName() string {
	return fmt.Sprintf("%s%s", jobGeneratedJobPrefix, j.Name())
}

// ReplaceImage For Job, we can't replace the Job' image
// but create a job with dev container instead
func (j *JobController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	j.Client.Context(ctx)
	originJob, err := j.Client.GetJobs(j.Name())
	if err != nil {
		return err
	}

	// Create a Job from origin Job's spec
	generatedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   j.getGeneratedJobName(),
			Labels: map[string]string{_const.DevWorkloadIgnored: "true"},
		},
		Spec: batchv1.JobSpec{
			Selector: originJob.Spec.Selector,
			Template: originJob.Spec.Template,
		},
	}

	if generatedJob.Spec.Template.ObjectMeta.Labels != nil {
		delete(generatedJob.Spec.Template.ObjectMeta.Labels, "controller-uid")
		delete(generatedJob.Spec.Template.ObjectMeta.Labels, "job-name")
	}
	if generatedJob.Spec.Selector != nil && generatedJob.Spec.Selector.MatchLabels != nil {
		delete(generatedJob.Spec.Selector.MatchLabels, "controller-uid")
	}

	devContainer, err := findContainerInJobSpec(generatedJob, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		j.genContainersAndVolumes(devContainer, ops.Container, ops.DevImage, ops.StorageClass)
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

func (j *JobController) Name() string {
	return j.Controller.Name
}

func (j *JobController) RollBack(reset bool) error {
	return j.Client.DeleteJob(j.getGeneratedJobName())
}

// GetPodList
// In DevMode, return pod list of generated Job.
// Otherwise, return pod list of original Job
func (j *JobController) GetPodList() ([]corev1.Pod, error) {
	if j.IsInDevMode() {
		pl, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
		if err != nil {
			return nil, err
		}
		return pl.Items, nil
	}
	pl, err := j.Client.ListPodsByJob(j.Name())
	if err != nil {
		return nil, err
	}
	return pl.Items, nil
}

func findContainerInJobSpec(job *batchv1.Job, containerName string) (*corev1.Container, error) {
	var devContainer *corev1.Container

	if containerName != "" {
		for index, c := range job.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				return &job.Spec.Template.Spec.Containers[index], nil
			}
		}
		return nil, errors.New(fmt.Sprintf("Container %s not found", containerName))
	} else {
		if len(job.Spec.Template.Spec.Containers) > 1 {
			return nil, errors.New(
				fmt.Sprintf(
					"There are more than one container defined," +
						"please specify one to start developing",
				),
			)
		}
		if len(job.Spec.Template.Spec.Containers) == 0 {
			return nil, errors.New("No container defined ???")
		}
		devContainer = &job.Spec.Template.Spec.Containers[0]
	}
	return devContainer, nil
}
