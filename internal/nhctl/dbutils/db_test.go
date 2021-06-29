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

package dbutils

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
	"testing"
	"time"
)

func TestOpenLevelDB(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{
		ErrorIfMissing: true,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("%v exists\n", db)
	}
}

func TestOpenLevelDBForPut(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("db opened\n")
	}

	time.Sleep(60 * time.Second)
	fmt.Println("After 60s")
	err = db.Put([]byte("aaa"), []byte("bbb"), nil)
	if err != nil {
		panic(err)
	}
}

func TestOpenLevelDBForLog(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{})
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	fmt.Printf("db opened111\n")

	fmt.Printf("in for %d", 1)
	for i := 0; i < 100; i++ {
		fmt.Println("Update ", i)
		err = db.Put([]byte("aaa"), []byte(fmt.Sprintf("bbb %d", i)), nil)
		if err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
	}
	defer db.Close()
}

func TestOpenLevelDBForIter(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ReadOnly: true})
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	defer db.Close()
	iter := db.NewIterator(nil, nil)
	if iter.Next() {
		fmt.Println(iter.Key())
	}
}

func TestOpenLevelDBForGetting(t *testing.T) {
	db, err := leveldb.OpenFile("/tmp/tmp/db", &opt.Options{ErrorIfMissing: true})
	if err != nil {
		panic(err)
	}
	v, err := db.GetProperty("leveldb.stats")
	if err != nil {
		panic(err)
	}
	fmt.Printf("leveldb.stats: %s\n", v)

	v, err = db.GetProperty("leveldb.num-files-at-level0")
	if err != nil {
		panic(err)
	}
	fmt.Printf("leveldb.num-files-at-level0: %s\n", v)

	v, err = db.GetProperty("leveldb.iostats")
	if err != nil {
		panic(err)
	}
	fmt.Printf("leveldb.iostats: %s\n", v)

	v, err = db.GetProperty("leveldb.writedelay")
	if err != nil {
		panic(err)
	}
	fmt.Printf("leveldb.writedelay: %s\n", v)

	v, err = db.GetProperty("leveldb.sstables")
	if err != nil {
		panic(err)
	}
	fmt.Printf("leveldb.sstables: %s\n", v)

}

func TestOpenLevelDBForCompact(t *testing.T) {

	db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ErrorIfMissing: true})
	if err != nil {
		panic(err)
	}
	// key: nh6lmaa.bookinfo.profile.v2
	s, err := db.SizeOf([]util.Range{*util.BytesPrefix([]byte("nh6lmaa.bookinfo.profile.v2"))})
	if err != nil {
		panic(err)
	}
	fmt.Println(s.Sum())

	//db.CompactRange()

	time.Sleep(1 * time.Second)
	db.Close()

}

func TestOpenLevelDBForOpenManyTime(t *testing.T) {

	for i := 0; i < 20; i++ {
		db, err := leveldb.OpenFile("/tmp/tmp/db2", &opt.Options{ErrorIfMissing: false, ReadOnly: false})
		if err != nil {
			panic(err)
		}
		bytes := []byte(strconv.Itoa(i))
		err = db.Put(bytes, bytes, nil)
		if err != nil {
			//fmt.Println(err)
		}
		db.Close()
	}
}
