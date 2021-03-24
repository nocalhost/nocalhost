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
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	leveldb_errors "github.com/syndtr/goleveldb/leveldb/errors"
	"nocalhost/pkg/nhctl/log"
	"syscall"
	"time"
)

const (
	DefaultApplicationDbDir = "db"
)

func openApplicationLevelDB(ns, app string, readonly bool) (*leveldb.DB, error) {
	existed := CheckIfApplicationExist(app, ns)
	if !existed {
		return nil, errors.New(fmt.Sprintf("Applicaton %s in %s not exists", app, ns))
	}

	path := getAppDbDir(ns, app)
	var o *opt.Options
	if readonly {
		o = &opt.Options{
			ReadOnly: true,
		}
	}
	db, err := leveldb.OpenFile(path, o)
	if err != nil {
		for i := 0; i < 300; i++ {
			if errors.Is(err, syscall.EAGAIN) {
				log.Log("Another process is accessing leveldb, wait for 0.2s to retry")
				time.Sleep(200 * time.Millisecond)
				db, err = leveldb.OpenFile(path, nil)
				if err == nil {
					break
				}
				//} else if strings.Contains(err.Error(), "leveldb: manifest corrupted") {
			} else if leveldb_errors.IsCorrupted(err) {
				log.Info("Recovering leveldb file...")
				db, err = leveldb.RecoverFile(path, nil)
				if err != nil {
					log.WarnE(err, "")
				} else {
					break
				}
			}
		}
		if err == nil {
			log.Log("Retry success")
		} else {
			return nil, errors.Wrap(err, "Retry opening leveldb failed : timeout")
		}
	}
	return db, nil
}

func profileV2Key(ns, app string) string {
	return fmt.Sprintf("%s.%s.profile.v2", ns, app)
}

func ListAllFromApplicationDb(ns, appName string) (map[string]string, error) {
	db, err := openApplicationLevelDB(ns, appName, true)
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
