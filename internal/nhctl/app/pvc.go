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

package app

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nhctl/log"
)

// Get all PersistVolumeClaims created by this application
func (a *Application) GetAllPVCs() ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(context.TODO(), a.GetNamespace(), map[string]string{AppLabel: a.Name})
}

// Get all PersistVolumeClaims created by specified service
func (a *Application) GetPVCsBySvc(svcName string) ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(context.TODO(), a.GetNamespace(), map[string]string{AppLabel: a.Name, ServiceLabel: svcName})
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
		err = a.client.DeletePVC(a.GetNamespace(), pvc.Name)
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
