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
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

type PVCFlags struct {
	App  string
	Svc  string
	Name string
	Yaml bool
	Json bool
}

var pvcFlags = PVCFlags{}

func init() {
	pvcListCmd.Flags().StringVar(&pvcFlags.App, "app", "", "List PVCs of specified application")
	pvcListCmd.Flags().StringVar(&pvcFlags.Svc, "svc", "", "List PVCs of specified service")
	pvcListCmd.Flags().BoolVar(&pvcFlags.Yaml, "yaml", false, "Use yaml as the output format")
	pvcListCmd.Flags().BoolVar(&pvcFlags.Json, "json", false, "Use json as the output format")
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
			//if !nocalhost.CheckIfApplicationExist(pvcFlags.App, nameSpace) {
			//	log.Fatalf("Application %s not found", pvcFlags.App)
			//}
			//nhApp, err := app.NewApplication(pvcFlags.App)
			//if err != nil {
			//	log.Fatalf("Failed to create application %s", pvcFlags.App)
			//}
			var err error
			initApp(pvcFlags.App)
			if pvcFlags.Svc != "" {
				exist, err := nocalhostApp.CheckIfSvcExist(pvcFlags.Svc, app.Deployment)
				if err != nil {
					log.FatalE(err, "failed to check if svc exists")
				} else if !exist {
					log.Fatalf("\"%s\" not found", pvcFlags.Svc)
				}
				pvcList, err = nocalhostApp.GetPVCsBySvc(pvcFlags.Svc)
				if err != nil {
					log.FatalE(err, "Failed to get PVCs")
				}
			} else {
				pvcList, err = nocalhostApp.GetAllPVCs()
				if err != nil {
					log.FatalE(err, "Failed to get PVCs")
				}
			}
		}

		if pvcFlags.Yaml {
			DisplayPVCsByYaml(pvcList)
		} else if pvcFlags.Json {
			DisplayPVCsByJson(pvcList)
		} else {
			DisplayPVCs(pvcList)
		}
	},
}

type pvcObject struct {
	Name         string `json:"name"          yaml:"name"`
	AppName      string `json:"app_name"      yaml:"appName"`
	ServiceName  string `json:"service_name"  yaml:"serviceName"`
	Capacity     string `json:"capacity"      yaml:"capacity"`
	StorageClass string `json:"storage_class" yaml:"storageClass"`
	Status       string `json:"status"        yaml:"status"`
	MountPath    string `json:"mount_path"    yaml:"mountPath"`
}

func makePVCObjectList(pvcList []v1.PersistentVolumeClaim) []*pvcObject {
	pvcObjectList := make([]*pvcObject, 0)

	for _, pvc := range pvcList {
		labels := pvc.Labels
		quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		annotations := pvc.Annotations
		pY := &pvcObject{
			Name:        pvc.Name,
			AppName:     labels[app.AppLabel],
			ServiceName: labels[app.ServiceLabel],
			Capacity:    quantity.String(),
			Status:      string(pvc.Status.Phase),
			MountPath:   annotations[app.PersistentVolumeDirLabel],
		}
		if pvc.Spec.StorageClassName != nil {
			pY.StorageClass = *pvc.Spec.StorageClassName
		}
		pvcObjectList = append(pvcObjectList, pY)
		//fmt.Printf("%s %s %s %s %s\n", pvc.Name, labels[app.AppLabel], labels[app.ServiceLabel], quantity.String(), pvc.Status.Phase)
	}

	return pvcObjectList
}

func DisplayPVCsByYaml(pvcList []v1.PersistentVolumeClaim) {
	pvcObjectList := makePVCObjectList(pvcList)
	bys, err := yaml.Marshal(pvcObjectList)
	if err != nil {
		log.FatalE(err, "fail to marshal")
	}
	fmt.Print(string(bys))
}

func DisplayPVCsByJson(pvcList []v1.PersistentVolumeClaim) {
	pvcObjectList := makePVCObjectList(pvcList)
	bys, err := json.Marshal(pvcObjectList)
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
		fmt.Printf(
			"%s %s %s %s %s\n",
			pvc.Name,
			labels[app.AppLabel],
			labels[app.ServiceLabel],
			quantity.String(),
			pvc.Status.Phase,
		)
	}
}
