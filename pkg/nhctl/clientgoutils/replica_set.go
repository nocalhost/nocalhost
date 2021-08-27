/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strconv"
	"time"
)

func (c *ClientGoUtils) UpdateReplicaSet(rs *v1.ReplicaSet) (*v1.ReplicaSet, error) {
	rs2, err := c.ClientSet.AppsV1().ReplicaSets(c.namespace).Update(c.ctx, rs, metav1.UpdateOptions{})
	return rs2, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetSortedReplicaSetsByDeployment(deployment string) ([]*v1.ReplicaSet, error) {
	rss, err := c.GetReplicaSetsByDeployment(deployment)
	if err != nil {
		return nil, err
	}
	if rss == nil || len(rss) < 1 {
		return nil, nil
	}
	keys := make([]int, 0)
	for rs := range rss {
		keys = append(keys, rs)
	}
	sort.Ints(keys)
	results := make([]*v1.ReplicaSet, 0)
	for _, key := range keys {
		results = append(results, rss[key])
	}
	return results, nil
}

func (c *ClientGoUtils) GetReplicaSetsByDeployment(deploymentName string) (map[int]*v1.ReplicaSet, error) {
	var rsList *v1.ReplicaSetList
	replicaSetsClient := c.ClientSet.AppsV1().ReplicaSets(c.namespace)
	rsList, err := replicaSetsClient.List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	rsMap := make(map[int]*v1.ReplicaSet)
	for _, item := range rsList.Items {
		if item.OwnerReferences == nil {
			continue
		}
		for _, owner := range item.OwnerReferences {
			if owner.Name == deploymentName && item.Annotations["deployment.kubernetes.io/revision"] != "" {
				if revision, err := strconv.Atoi(item.Annotations["deployment.kubernetes.io/revision"]); err == nil {
					rsMap[revision] = item.DeepCopy()
				}
			}
		}
	}
	return rsMap, nil
}

// WaitLatestRevisionReady make sure other revision's replicas is 0
// For deployment to enter dev mode
func (c *ClientGoUtils) WaitLatestRevisionReady(deploymentName string) error {

	cancelCtx, cancelFun := context.WithCancel(context.Background())
	errChan := make(chan error, 0)
	go func() {
		printed := false
		for {
			select {
			case <-cancelCtx.Done():
				return
			default:
				<-time.Tick(2 * time.Second)

				deploy, err := c.GetDeployment(deploymentName)
				if err != nil {
					errChan <- err
					return
				}

				// Check if deployment's condition is FailedCreate
				replicaFailure, _, failMess, _ := CheckIfDeploymentIsReplicaFailure(deploy)
				if replicaFailure {
					errChan <- errors.New(fmt.Sprintf("deployment is in ReplicaFailure condition - %s", failMess))
					return
				}

				replicaSets, err := c.GetReplicaSetsByDeployment(deploymentName)
				if err != nil {
					errChan <- err
					return
				}

				revisions := make([]int, 0)
				for _, rs := range replicaSets {
					if rs.Annotations["deployment.kubernetes.io/revision"] != "" {
						r, _ := strconv.Atoi(rs.Annotations["deployment.kubernetes.io/revision"])
						revisions = append(revisions, r)
					}
				}
				sort.Ints(revisions)
				latestRevision := revisions[len(revisions)-1]

				isReady := true
				for _, rs := range replicaSets {
					if rs.Annotations["deployment.kubernetes.io/revision"] == strconv.Itoa(latestRevision) {
						// Check replicaSet's events
						events, err := c.ListEventsByReplicaSet(rs.Name)
						if err != nil || len(events) == 0 {
							continue
						}

						for _, event := range events {
							if event.Type == "Warning" {
								if event.Reason == "FailedCreate" {
									errChan <- errors.New(fmt.Sprintf("Latest ReplicaSet failed to be ready - %s", event.Message))
									return
								}
								log.Warnf("Warning event: %s", event.Message)
							}
						}
						continue
					}
					if rs.Status.Replicas != 0 {
						if !printed {
							printed = true
							log.Infof(
								"Previous replicaSet %s has not been terminated, waiting revision %d to be ready",
								rs.Name,
								latestRevision,
							)
							log.Info("This may take several minutes, depending on the load of your k8s cluster")
						}
						isReady = false
						break
					}
				}
				if isReady {
					errChan <- nil
					return
				}
			}
		}
	}()

	for i := 1; i <= 30; i++ {
		select {
		case err := <-errChan:
			return err
		case <-time.After(1 * time.Minute):
			pods, err := c.ListLatestRevisionPodsByDeployment(deploymentName)
			if err != nil {
				log.WarnE(err, "Failed to list latest revision")
				continue
			}
			if len(pods) != 1 {
				log.Warnf("The number of pods is not 1 ?(but %d)", len(pods))
				continue
			}
			pod := pods[0]
			conStr := ""
			warnMsg := ""
			msg := ""
			for _, con := range pod.Status.Conditions {
				if con.Type == "PodScheduled" && con.Status != "True" {
					conStr = fmt.Sprintf("%s Reason: %s, Type: %s", con.Message, con.Reason, con.Type)
					msg = con.Message
				}
			}
			if conStr != "" {
				warnMsg = fmt.Sprintf("The phase of %s is %s. %s", pod.Name, pod.Status.Phase, conStr)
				if i == 5 {
					log.PWarnf("Entering DevMode has spent more than 5 minutes. If you want to exit it, please click the button in the bottom of the IDE. Status: %s", msg)
					continue
				}
			} else {
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.Ready == true {
						continue
					}
					if cs.State.Waiting != nil {
						warnMsg = fmt.Sprintf("%s Reason: %s [Container %s]",
							cs.State.Waiting.Message, cs.State.Waiting.Reason, cs.Name)
						msg = cs.State.Waiting.Message
					}
				}
				if i == 5 {
					log.PWarnf("Entering DevMode has spent more than 5 minutes. If you want to exit it, please click the button in the bottom of the IDE. Status: %s", msg)
					continue
				}
			}
			if warnMsg != "" {
				log.Infof("%s - [Pod %s]", warnMsg, pod.Name)
			}
		}
	}
	cancelFun()
	return errors.New("Waiting new revision ReplicaSets timeout")
}
