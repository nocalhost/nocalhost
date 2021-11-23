/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"time"
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
	return fmt.Sprintf("%s%s", jobGeneratedJobPrefix, j.GetName())
}

// ReplaceImage For Job, we can't replace the Job' image
// but create a job with dev container instead
func (j *JobController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	j.Client.Context(ctx)
	originJob, err := j.Client.GetJobs(j.GetName())
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

	devContainer, sideCarContainer, devModeVolumes, err :=
		j.genContainersAndVolumes(&generatedJob.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, false)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&generatedJob.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	// Create generated deployment
	if _, err = j.Client.CreateJob(generatedJob); err != nil {
		return err
	}

	j.patchAfterDevContainerReplaced(ops.Container, generatedJob.Kind, generatedJob.Name)
	<-time.Tick(time.Second)

	return waitingPodToBeReady(j.GetNocalhostDevContainerPod)
}

func (j *JobController) RollBack(reset bool) error {
	return j.Client.DeleteJob(j.getGeneratedJobName())
}

// GetPodList
// In DevMode, return pod list of generated Job.
// Otherwise, return pod list of original Job
func (j *JobController) GetPodList() ([]corev1.Pod, error) {
	if j.IsInReplaceDevMode() {
		pl, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
		if err != nil {
			return nil, err
		}
		return pl.Items, nil
	}
	pl, err := j.Client.ListPodsByJob(j.GetName())
	if err != nil {
		return nil, err
	}
	return pl.Items, nil
}
