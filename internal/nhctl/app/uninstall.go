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

import "nocalhost/pkg/nhctl/log"

func (a *Application) cleanUpDepConfigMap() error {

	if a.AppProfile.DependencyConfigMapName != "" {
		log.Debugf("delete config map %s", a.AppProfile.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(a.AppProfile.DependencyConfigMapName, a.AppProfile.Namespace)
		if err != nil {
			return err
		}
		a.AppProfile.DependencyConfigMapName = ""
		a.AppProfile.Save()
	}
	return nil
}
