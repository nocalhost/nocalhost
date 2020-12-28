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

package cmds

import (
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/app"
)

type PvcList struct {
}

type NocalhostPvc struct {
	Name         string
	Application  string
	Service      string
	Capacity     string
	Status       string
	StorageClass string
}

func NewNocalhostPvc(pvc *v1.PersistentVolumeClaim) *NocalhostPvc {
	if pvc == nil {
		return nil
	}

	labels := pvc.Labels
	quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	nhPvc := &NocalhostPvc{
		Name:        pvc.Name,
		Application: labels[app.AppLabel],
		Service:     labels[app.ServiceLabel],
		Capacity:    quantity.String(),
		Status:      string(pvc.Status.Phase),
	}
	if pvc.Spec.StorageClassName != nil {
		nhPvc.StorageClass = *pvc.Spec.StorageClassName
	}

	return nhPvc
}
