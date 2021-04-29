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
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
)

func (c *Controller) RollBack(reset bool) error {
	clientUtils := c.Client

	switch c.Type {
	case appmeta.Deployment:
		dep, err := clientUtils.GetDeployment(c.Name)
		if err != nil {
			return err
		}

		rss, err := clientUtils.GetSortedReplicaSetsByDeployment(c.Name)
		if err != nil {
			log.WarnE(err, "Failed to get rs list")
			return err
		}

		// Find previous replicaSet
		if len(rss) < 2 {
			log.Warn("No history to roll back")
			return nil
		}

		var r *v1.ReplicaSet
		var originalPodReplicas *int32
		for _, rs := range rss {
			if rs.Annotations == nil {
				continue
			}
			// Mark the original revision
			if rs.Annotations[nocalhost.DevImageRevisionAnnotationKey] == nocalhost.DevImageRevisionAnnotationValue {
				r = rs
				if rs.Annotations[nocalhost.DevImageOriginalPodReplicasAnnotationKey] != "" {
					podReplicas, _ := strconv.Atoi(rs.Annotations[nocalhost.DevImageOriginalPodReplicasAnnotationKey])
					podReplicas32 := int32(podReplicas)
					originalPodReplicas = &podReplicas32
				}
			}
		}
		if r == nil {
			if !reset {
				return errors.New("Failed to find the proper revision to rollback")
			} else {
				r = rss[0]
			}
		}

		dep.Spec.Template = r.Spec.Template
		if originalPodReplicas != nil {
			dep.Spec.Replicas = originalPodReplicas
		}

		log.Info(" Deleting current revision...")
		err = clientUtils.DeleteDeployment(dep.Name, false)
		if err != nil {
			return err
		}

		log.Info(" Recreating original revision...")
		dep.ResourceVersion = ""
		if len(dep.Annotations) == 0 {
			dep.Annotations = make(map[string]string, 0)
		}
		dep.Annotations["nocalhost-dep-ignore"] = "true"

		// Add labels and annotations
		if dep.Labels == nil {
			dep.Labels = make(map[string]string, 0)
		}
		dep.Labels[nocalhost.AppManagedByLabel] = nocalhost.AppManagedByNocalhost

		if dep.Annotations == nil {
			dep.Annotations = make(map[string]string, 0)
		}
		dep.Annotations[nocalhost.NocalhostApplicationName] = c.AppName
		dep.Annotations[nocalhost.NocalhostApplicationNamespace] = c.NameSpace

		_, err = clientUtils.CreateDeployment(dep)
		if err != nil {
			if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
				log.Warn("[Warning] Nocalhost-dep needs to update")
			}
			return err
		}
	default:
		return errors.New(fmt.Sprintf("%s has not been supported yet", c.Type))
	}
	return nil
}

func (c *Controller) DevEnd(reset bool) error {
	if err := c.RollBack(reset); err != nil {
		if !reset {
			return err
		}
		log.WarnE(err, "something incorrect occurs when rolling back")
	}

	utils.ShouldI(c.AppMeta.SvcDevEnd(c.Name, c.Type), "something incorrect occurs when updating secret")
	utils.ShouldI(c.StopSyncAndPortForwardProcess(true), "something incorrect occurs when stopping sync process")
	return nil
}
