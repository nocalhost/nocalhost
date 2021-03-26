/*
Copyright 2021 The Nocalhost Authors.
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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/profile"
)

func (a *Application) CheckIfPortForwardExists(svcName string, localPort, remotePort int) bool {

	_ = a.LoadAppProfileV2()

	for _, portForward := range a.GetSvcProfileV2(svcName).DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			return true
		}
	}
	return false
}

//func (a *Application) StartTransaction() error {
//	var err error
//	if a.db != nil {
//		return errors.New("Transaction already start")
//	}
//	a.db, err = nocalhost.OpenApplicationLevelDB(a.NameSpace, a.Name, false)
//	return err
//}
//
//func (a *Application) StopTransaction() error {
//	var err error
//	if a.db != nil {
//		err = a.db.Close()
//		a.db = nil
//	}
//	return err
//}

// You should `CheckIfPortForwardExists` before adding a port-forward to db
func (a *Application) AddPortForwardToDB(svcName string, port *profile.DevPortForward) error {
	//db, err := nocalhost.OpenApplicationLevelDB(a.NameSpace, a.Name, false)
	//if err != nil {
	//	return err
	//}
	//defer db.Close()
	//
	//profileV2, err := nocalhost.GetProfileV2(a.NameSpace, a.Name, db)
	//if err != nil {
	//	return err
	//}

	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name)
	if err != nil {
		return err
	}

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to add a port-forward to db")
	}

	svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, port)
	return profileV2.SaveAndCloseDb()
}

func (a *Application) DeletePortForwardFromDB(svcName string, localPort, remotePort int) error {
	//db, err := nocalhost.OpenApplicationLevelDB(a.NameSpace, a.Name, false)
	//if err != nil {
	//	return err
	//}
	//defer db.Close()
	//
	//profileV2, err := nocalhost.GetProfileV2(a.NameSpace, a.Name, db)
	//if err != nil {
	//	return err
	//}
	//
	//svcProfile := fetchSvcProfileV2FromProfile(svcName, profileV2)
	//if svcProfile == nil {
	//	return errors.New("Failed to add a port-forward to db")
	//}

	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name)
	if err != nil {
		return err
	}

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
		return profileV2.SaveAndCloseDb()
	}
	return nil
}
