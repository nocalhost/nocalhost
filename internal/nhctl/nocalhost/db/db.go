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

package db

import (
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
)

func OpenApplicationLevelDB(ns, app string, readonly bool) (*dbutils.LevelDBUtils, error) {
	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.OpenLevelDB(path, readonly)
}

func CreateApplicationLevelDB(ns, app string, errorIfExist bool) error {
	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.CreateLevelDB(path, errorIfExist)
}
