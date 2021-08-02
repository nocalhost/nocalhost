/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
)

// Get all PersistVolumeClaims created by this application
func (a *Application) GetAllPVCs() ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(map[string]string{_const.AppLabel: a.Name})
}

// GetPVCsBySvc todo hxx: move to controller
// Get all PersistVolumeClaims created by specified service
func (a *Application) GetPVCsBySvc(svcName string) ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(map[string]string{_const.AppLabel: a.Name, _const.ServiceLabel: svcName})
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
		log.Info("No Persistent volume needs to be cleaned up")
		return nil
	}

	// todo check if pvc still is used by some pods
	for _, pvc := range pvcs {
		err = a.client.DeletePVC(pvc.Name)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("error occurs while deleting persistent volume %s", pvc.Name))
			if !continueOnErr {
				return err
			}
		} else {
			log.Infof("Persistent volume %s has been cleaned up", pvc.Name)
		}
	}
	return nil
}
