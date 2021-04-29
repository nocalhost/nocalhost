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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) CheckIfPortForwardExists(svcName string, localPort, remotePort int) (bool, error) {
	svcProfile, err := a.GetSvcProfile(svcName)
	if err != nil {
		return false, err
	}
	for _, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			return true, nil
		}
	}
	return false, nil
}

// You should `CheckIfPortForwardExists` before adding a port-forward to db
func (a *Application) AddPortForwardToDB(svcName string, port *profile.DevPortForward) error {

	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}

	svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, port)
	return profileV2.Save()
}

func (a *Application) DeletePortForwardFromDB(svcName string, localPort, remotePort int) error {

	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to add a port-forward to db")
	}

	indexToDelete := -1
	for index, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			indexToDelete = index
			break
		}
	}

	// remove portForward from DevPortForwardList
	if indexToDelete > -1 {
		originList := svcProfile.DevPortForwardList
		newList := make([]*profile.DevPortForward, 0)
		for index, p := range originList {
			if index != indexToDelete {
				newList = append(newList, p)
			}
		}
		svcProfile.DevPortForwardList = newList
		log.Logf("Deleting pf %d:%d", localPort, remotePort)
		//for _, pf := range profileV2.FetchSvcProfileV2FromProfile(svcName).DevPortForwardList {
		//	log.Infof("%v", *pf)
		//}
		return profileV2.Save()
	}
	return nil
}
