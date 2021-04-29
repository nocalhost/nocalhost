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
	"fmt"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

// Get all PersistVolumeClaims created by this application
func (a *Application) GetAllPVCs() ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(map[string]string{nocalhost.AppLabel: a.Name})
}

// GetPVCsBySvc todo hxx: move to controller
// Get all PersistVolumeClaims created by specified service
func (a *Application) GetPVCsBySvc(svcName string) ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(map[string]string{nocalhost.AppLabel: a.Name, nocalhost.ServiceLabel: svcName})
}

// If svcName specified, cleaning pvcs created by the service
// If svcName not specified, cleaning all pvcs created by the application
func (a *Application) CleanUpPVCs(svcName string, continueOnErr bool) error {
	var (
		pvcs []v1.PersistentVolumeClaim
		err  error
	)
	if svcName == "" {
		pvcs, err = a.GetAllPVCs()
	} else {
		pvcs, err = a.GetPVCsBySvc(svcName)
	}
	if err != nil {
		return err
	}
	if len(pvcs) == 0 {
		log.Infof("No pvc need to be cleaned up")
		return nil
	}

	// todo check if pvc still is used by some pods
	for _, pvc := range pvcs {
		err = a.client.DeletePVC(pvc.Name)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("error occurs while deleting pvc %s", pvc.Name))
			if !continueOnErr {
				return err
			}
		} else {
			log.Infof("Pvc %s cleaned up", pvc.Name)
		}
	}
	return nil
}

func (a *Application) CleanUpPVC(name string) error {
	return a.client.DeletePVC(name)
}
