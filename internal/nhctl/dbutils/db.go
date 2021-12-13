/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dbutils

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_errors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strconv"
	"syscall"
	"time"
)

// Initial a level db
// If leveldb already exists, return error
func CreateLevelDB(path string, errorIfExist bool) error {
	db, err := leveldb.OpenFile(path, &opt.Options{
		ErrorIfExist: errorIfExist,
	})
	if db != nil {
		_ = db.Close()
	}
	return errors.Wrap(err, "")
}

func CreateLevelDBWithInitData(path, key string, data []byte, errorIfExist bool) error {
	db, err := leveldb.OpenFile(path, &opt.Options{
		ErrorIfExist: errorIfExist,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	defer db.Close()

	return errors.WithStack(db.Put([]byte(key), data, nil))
}

// Open a leveldb
// If leveldb is corrupted, try to recover it
// If leveldb is EAGAIN, retry to open it in 1 minutes
// If leveldb is missing, return a error instead create one
func OpenLevelDB(path string, readonly bool) (*LevelDBUtils, error) {
	var o *opt.Options
	o = &opt.Options{
		ErrorIfMissing: true,
	}
	if readonly {
		o.ReadOnly = true
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
	}
	db, err := leveldb.OpenFile(path, o)
	if err != nil {
		if leveldb_errors.IsCorrupted(err) {
			log.Log("Recovering leveldb file...")
			db, err = leveldb.RecoverFile(path, nil)
		} else if errors.Is(err, syscall.ENOENT) || os.IsNotExist(err) {
			return nil, errors.Wrap(err, "File not exist, not need to retry")
		} else /*if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EBUSY)*/ {
			log.Logf("Another process is accessing leveldb: %s, wait for 0.002s to retry, err: %v", path, err)
			for i := 0; i < 3000; i++ {
				time.Sleep(10 * time.Millisecond)
				db, err = leveldb.OpenFile(path, o)
				if err == nil {
					log.Logf("Success after %v times, open leveldb: %s", i, path)
					break
				}
			}
		}
		if err != nil {
			return nil, errors.Wrap(err, "Retry opening leveldb failed")
		}
	}

	dbUtils := &LevelDBUtils{
		readonly: readonly,
		db:       db,
	}

	if !readonly {
		v, err := db.GetProperty("leveldb.num-files-at-level0")
		if err != nil {
			log.LogE(err)
		} else {
			num, err := strconv.Atoi(v)
			if err != nil {
				log.LogE(err)
			}
			if num > 10 {
				log.Logf("Compacting %s", path)
				if err = dbUtils.CompactFirstKey(); err != nil {
					log.LogE(err)
				}
			}
		}
	}

	return dbUtils, nil
}
