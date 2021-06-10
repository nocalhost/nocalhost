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

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/clientgoutils"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"

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
			var err error
			initApp(pvcFlags.App)
			if pvcFlags.Svc != "" {
				exist, err := nocalhostApp.Controller(pvcFlags.Svc, base.Deployment).CheckIfExist()
				if err != nil {
					log.FatalE(err, "failed to check if controller exists")
				} else if !exist {
					log.Fatalf("\"%s\" not found", pvcFlags.Svc)
				}
				pvcList, err = nocalhostApp.GetPVCsBySvc(pvcFlags.Svc)
				must(err)
			} else {
				pvcList, err = nocalhostApp.GetAllPVCs()
				must(err)
			}
		} else {
			// List all pvc of current namespace
			cli, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
			must(err)
			pvcList, err = cli.ListPvcs()
			must(err)
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
	Name         string `json:"name" yaml:"name"`
	AppName      string `json:"app_name" yaml:"appName"`
	ServiceName  string `json:"service_name" yaml:"serviceName"`
	Capacity     string `json:"capacity" yaml:"capacity"`
	StorageClass string `json:"storage_class" yaml:"storageClass"`
	Status       string `json:"status" yaml:"status"`
	MountPath    string `json:"mount_path" yaml:"mountPath"`
}

func makePVCObjectList(pvcList []v1.PersistentVolumeClaim) []*pvcObject {
	pvcObjectList := make([]*pvcObject, 0)

	for _, pvc := range pvcList {
		labels := pvc.Labels
		quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		annotations := pvc.Annotations
		pY := &pvcObject{
			Name:        pvc.Name,
			AppName:     labels[nocalhost.AppLabel],
			ServiceName: labels[nocalhost.ServiceLabel],
			Capacity:    quantity.String(),
			Status:      string(pvc.Status.Phase),
			MountPath:   annotations[nocalhost.PersistentVolumeDirLabel],
		}
		if pvc.Spec.StorageClassName != nil {
			pY.StorageClass = *pvc.Spec.StorageClassName
		}
		pvcObjectList = append(pvcObjectList, pY)
	}

	return pvcObjectList
}

func DisplayPVCsByYaml(pvcList []v1.PersistentVolumeClaim) {
	pvcObjectList := makePVCObjectList(pvcList)
	bys, err := yaml.Marshal(pvcObjectList)
	must(errors.Wrap(err, ""))
	fmt.Print(string(bys))
}

func DisplayPVCsByJson(pvcList []v1.PersistentVolumeClaim) {
	pvcObjectList := makePVCObjectList(pvcList)
	bys, err := json.Marshal(pvcObjectList)
	must(errors.Wrap(err, ""))
	fmt.Print(string(bys))
}

func DisplayPVCs(pvcList []v1.PersistentVolumeClaim) {
	fmt.Println("PVC AppName ServiceName Capacity Status")
	for _, pvc := range pvcList {
		labels := pvc.Labels
		quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		fmt.Printf(
			"%s %s %s %s %s\n", pvc.Name, labels[nocalhost.AppLabel], labels[nocalhost.ServiceLabel], quantity.String(),
			pvc.Status.Phase,
		)
	}
}
