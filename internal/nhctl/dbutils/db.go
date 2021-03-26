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

package dbutils

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_errors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"math/rand"
	"nocalhost/pkg/nhctl/log"
	"syscall"
	"time"
)

func OpenApplicationLevelDB(path string, readonly bool) (*leveldb.DB, error) {
	//existed := CheckIfApplicationExist(app, ns)
	//if !existed {
	//	return nil, errors.New(fmt.Sprintf("Applicaton %s in %s not exists", app, ns))
	//}

	//path := getAppDbDir(ns, app)
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
				log.Logf("Another process is accessing leveldb, wait for 0.2s to retry %s %s",err.Error(), path)
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
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
