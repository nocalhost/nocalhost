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
	"github.com/syndtr/goleveldb/leveldb/util"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost_path"
)

func OpenApplicationLevelDB(ns, app string, readonly bool) (*leveldb.DB, error) {
	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.OpenLevelDB(path, readonly)
}

func CreateApplicationLevelDB(ns, app string) error {
	path := nocalhost_path.GetAppDbDir(ns, app)
	return dbutils.CreateLevelDB(path)
}

// todo
//func CheckIfApplicationLevelDBExists(ns,app string) error {
//	path := nocalhost_path.GetAppDbDir(ns, app)
//
//}

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
	return result, nil
}

func CompactApplicationDb(ns, appName, key string) error {
	db, err := OpenApplicationLevelDB(ns, appName, false)
	if err != nil {
		return err
	}
	defer db.Close()
	if key == "" {
		iter := db.NewIterator(nil, nil)
		keys := make([][]byte, 0)
		for iter.Next() {
			keys = append(keys, iter.Key())
		}
		iter.Release()
		if len(keys) == 0 {
			return errors.New("No key to compact!")
		}
		key = string(keys[0])
	}
	return db.CompactRange(*util.BytesPrefix([]byte(key)))
}

func GetApplicationDbSize(ns, appName string) (int, error) {
	db, err := OpenApplicationLevelDB(ns, appName, true)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	iter := db.NewIterator(nil, nil)
	keys := make([][]byte, 0)
	for iter.Next() {
		keys = append(keys, iter.Key())
	}
	iter.Release()
	ranges := make([]util.Range, 0)
	for _, key := range keys {
		ranges = append(ranges, *util.BytesPrefix(key))
	}
	s, err := db.SizeOf(ranges)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return int(s.Sum()), nil
}
