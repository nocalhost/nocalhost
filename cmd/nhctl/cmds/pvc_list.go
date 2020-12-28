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
	"fmt"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

type PVCFlags struct {
	App  string
	Svc  string
	Yaml bool
}

var pvcFlags = PVCFlags{}

func init() {
	pvcListCmd.Flags().StringVar(&pvcFlags.App, "app", "", "List PVCs of specified application")
	pvcListCmd.Flags().StringVar(&pvcFlags.Svc, "svc", "", "List PVCs of specified service")
	pvcListCmd.Flags().BoolVar(&pvcFlags.Yaml, "yaml", false, "Use yaml as the output format")
	pvcCmd.AddCommand(pvcListCmd)
}

var pvcListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List PersistVolumeClaims",
	Long:    `List PersistVolumeClaims`,
	Run: func(cmd *cobra.Command, args []string) {
		var pvcList []v1.PersistentVolumeClaim
		if pvcFlags.App != "" {
			if !nocalhost.CheckIfApplicationExist(pvcFlags.App) {
				log.Fatalf("Application %s not found", pvcFlags.App)
			}
			nhApp, err := app.NewApplication(pvcFlags.App)
			if err != nil {
				log.Fatalf("Failed to create application %s", pvcFlags.App)
			}
			if pvcFlags.Svc != "" {
				exist, err := nhApp.CheckIfSvcExist(pvcFlags.Svc, app.Deployment)
				if err != nil {
					log.FatalE(err, "failed to check if svc exists")
				} else if !exist {
					log.Fatalf("\"%s\" not found", pvcFlags.Svc)
				}
				pvcList, err = nhApp.GetPVCsBySvc(pvcFlags.Svc)
				if err != nil {
					log.FatalE(err, "Failed to get PVCs")
				}
			} else {
				pvcList, err = nhApp.GetAllPVCs()
				if err != nil {
					log.FatalE(err, "Failed to get PVCs")
				}
			}
		} else {
			// list all pvc
		}
		//if len(pvcList) == 0 {
		//	log.Info("No pvc found")
		//}

		if pvcFlags.Yaml {
			DisplayPVCsByYaml(pvcList)
		} else {
			DisplayPVCs(pvcList)
		}
	},
}

type pvcYaml struct {
	Name         string `json:"name" yaml:"name"`
	AppName      string `json:"app_name" yaml:"appName"`
	ServiceName  string `json:"service_name" yaml:"serviceName"`
	Capacity     string `json:"capacity" yaml:"capacity"`
	StorageClass string `json:"storage_class" yaml:"storageClass"`
	Status       string `json:"status" yaml:"status"`
}

func DisplayPVCsByYaml(pvcList []v1.PersistentVolumeClaim) {

	pvcYamlList := make([]*pvcYaml, 0)

	for _, pvc := range pvcList {
		labels := pvc.Labels
		quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		pY := &pvcYaml{
			Name:        pvc.Name,
			AppName:     labels[app.AppLabel],
			ServiceName: labels[app.ServiceLabel],
			Capacity:    quantity.String(),
			Status:      string(pvc.Status.Phase),
		}
		if pvc.Spec.StorageClassName != nil {
			pY.StorageClass = *pvc.Spec.StorageClassName
		}
		pvcYamlList = append(pvcYamlList, pY)
		//fmt.Printf("%s %s %s %s %s\n", pvc.Name, labels[app.AppLabel], labels[app.ServiceLabel], quantity.String(), pvc.Status.Phase)
	}
	bys, err := yaml.Marshal(pvcYamlList)
	if err != nil {
		log.FatalE(err, "fail to marshal")
	}
	fmt.Print(string(bys))
}

func DisplayPVCs(pvcList []v1.PersistentVolumeClaim) {
	fmt.Println("PVC AppName ServiceName Capacity Status")
	for _, pvc := range pvcList {
		labels := pvc.Labels
		quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		fmt.Printf("%s %s %s %s %s\n", pvc.Name, labels[app.AppLabel], labels[app.ServiceLabel], quantity.String(), pvc.Status.Phase)
	}
}
