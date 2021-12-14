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
	"nocalhost/pkg/nhctl/log"
	"strconv"
)

type CronJobController struct {
	*Controller
}

const (
	cronjobGeneratedJobPrefix = "cronjob-generated-job-"
	cronjobScheduleAnnotation = "nocalhost.dev.cronjob.schedule" // deprecated
	cronjobSuspendAnnotation  = "nocalhost.dev.cronjob.suspend"
)

func (j *CronJobController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList.Items)
}

func (j *CronJobController) getGeneratedJobName() string {
	return fmt.Sprintf("%s%s", cronjobGeneratedJobPrefix, j.GetName())
}

// ReplaceImage For Job, we can't replace the Job' image
// but create a job with dev container instead
func (j *CronJobController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	j.Client.Context(ctx)
	originJob, err := j.Client.GetCronJobs(j.GetName())
	if err != nil {
		return err
	}

	if originJob.Annotations == nil {
		originJob.Annotations = make(map[string]string, 0)
	}
	if _, ok := originJob.Annotations[cronjobScheduleAnnotation]; ok {
		log.Infof("Removing deprecated annotation %s", cronjobScheduleAnnotation)
		delete(originJob.Annotations, cronjobScheduleAnnotation)
	}
	//originJob.Annotations[cronjobScheduleAnnotation] = originJob.Spec.Schedule
	if originJob.Spec.Suspend != nil {
		originJob.Annotations[cronjobSuspendAnnotation] = fmt.Sprintf("%t", *originJob.Spec.Suspend)
	}
	//originJob.Spec.Schedule = "1 1 1 1 1"
	isSuspend := true
	originJob.Spec.Suspend = &isSuspend
	log.Info("Suspending cronjob...")
	if _, err = j.Client.UpdateCronJob(originJob); err != nil {
		return err
	}

	// Create a Job from origin Job's spec
	generatedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   j.getGeneratedJobName(),
			Labels: map[string]string{_const.DevWorkloadIgnored: "true"},
		},
		Spec: batchv1.JobSpec{
			Selector: originJob.Spec.JobTemplate.Spec.Selector,
			Template: originJob.Spec.JobTemplate.Spec.Template,
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

	return waitingPodToBeReady(j.GetNocalhostDevContainerPod)
}

func (j *CronJobController) RollBack(reset bool) error {
	originJob, err := j.Client.GetCronJobs(j.GetName())
	if err != nil {
		return err
	}
	schedule, ok := originJob.Annotations[cronjobScheduleAnnotation] // for compatibility
	if ok {
		originJob.Spec.Schedule = schedule
		log.Infof("Recover schedule to %s", schedule)
		if _, err = j.Client.UpdateCronJob(originJob); err != nil {
			return err
		}
	} else {
		s := false
		suspend, ok := originJob.Annotations[cronjobSuspendAnnotation]
		if ok {
			s, _ = strconv.ParseBool(suspend)
		}
		originJob.Spec.Suspend = &s
		if _, err = j.Client.UpdateCronJob(originJob); err != nil {
			return err
		}
	}
	if err = j.Client.DeleteJob(j.getGeneratedJobName()); err != nil {
		if !reset {
			return err
		}
		log.WarnE(err, "")
	}
	return nil
}

// GetPodList
// In DevMode, return pod list of generated Job.
// Otherwise, return pod list of original Job
func (j *CronJobController) GetPodList() ([]corev1.Pod, error) {
	if j.IsInReplaceDevMode() {
		pl, err := j.Client.ListPodsByJob(j.getGeneratedJobName())
		if err != nil {
			return nil, err
		}
		return pl.Items, nil
	}
	pl, err := j.Client.ListPodsByCronJob(j.GetName())
	if err != nil {
		return nil, err
	}
	return pl.Items, nil
}
