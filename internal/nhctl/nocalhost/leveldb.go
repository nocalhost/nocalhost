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

package nocalhost

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
)

func OpenApplicationLevelDB(ns, app string, readonly bool) (*leveldb.DB, error) {
	//existed := CheckIfApplicationExist(app, ns)
	//if !existed {
	//	return nil, errors.New(fmt.Sprintf("Applicaton %s in %s not exists", app, ns))
	//}

	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.OpenLevelDB(path, readonly)

}

func ListAllFromApplicationDb(ns, appName string) (map[string]string, error) {
	db, err := OpenApplicationLevelDB(ns, appName, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	result := make(map[string]string, 0)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		result[string(iter.Key())] = string(iter.Value())
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	//fmt.Println("Hold db lock for 1 minutes")
	//time.Sleep(1 * time.Minute)
	return result, nil
}
