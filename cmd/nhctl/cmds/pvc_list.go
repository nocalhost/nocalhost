/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/clientgoutils"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
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
	pvcListCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
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
			if pvcFlags.Svc != "" {
				_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(pvcFlags.App, pvcFlags.Svc, common.ServiceType)
				must(err)
				pvcList, err = nocalhostSvc.GetPVCsBySvc()
				must(err)
			} else {
				nocalhostApp, err := common.InitApp(pvcFlags.App)
				must(err)
				pvcList, err = nocalhostApp.GetAllPVCs()
				must(err)
			}
		} else {
			// List all pvc of current namespace
			cli, err := clientgoutils.NewClientGoUtils(common.KubeConfig, common.NameSpace)
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
			AppName:     labels[_const.AppLabel],
			ServiceName: labels[_const.ServiceLabel],
			Capacity:    quantity.String(),
			Status:      string(pvc.Status.Phase),
			MountPath:   annotations[_const.PersistentVolumeDirLabel],
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
			"%s %s %s %s %s\n", pvc.Name, labels[_const.AppLabel], labels[_const.ServiceLabel], quantity.String(),
			pvc.Status.Phase,
		)
	}
}
