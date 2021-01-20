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

package clientgoutils

import (
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

func (c *ClientGoUtils) WaitLatestRevisionReplicaSetOfDeploymentToBeReady(deploymentName string) error {

	for {
		time.Sleep(2 * time.Second)
		replicaSets, err := c.GetReplicaSetsByDeployment(deploymentName)
		if err != nil {
			log.WarnE(err, "Failed to get replica sets")
			return err
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
				// check pod's events
				events, err := c.ListEventsByReplicaSet(rs.Name)
				if err != nil || len(events) == 0 {
					continue
				}

				for _, event := range events {
					if event.Reason == "FailedCreate" && time.Now().Sub(event.LastTimestamp.Time).Minutes() < 1 {
						return errors.New(fmt.Sprintf("Latest ReplicaSet failed to be ready : %s", event.Message))
					}
				}
				continue
			}
			if rs.Status.Replicas != 0 {
				log.Infof("Previous replicaSet %s has not been terminated, waiting revision %d to be ready", rs.Name, latestRevision)
				isReady = false
				break
			}
		}
		if isReady {
			return nil
		}
	}
}
