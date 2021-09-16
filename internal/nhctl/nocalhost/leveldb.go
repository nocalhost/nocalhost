/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"github.com/pkg/errors"
	nocalhost_db "nocalhost/internal/nhctl/nocalhost/db"
)

func ListAllFromApplicationDb(ns, appName, nid string) (map[string]string, error) {
	db, err := nocalhost_db.OpenApplicationLevelDB(ns, appName, nid, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return db.ListAll()
}

func CompactApplicationDb(ns, appName, nid, key string) error {
	db, err := nocalhost_db.OpenApplicationLevelDB(ns, appName, nid, false)
	if err != nil {
		return err
	}
	defer db.Close()
	if key == "" {
		result, err := db.ListAll()
		if err != nil {
			return err
		}
		if len(result) == 0 {
			return errors.New("No key to compact!")
		}
		for k := range result {
			key = k // Get the first key
			break
		}
	}
	return db.CompactKey([]byte(key))
}

func GetApplicationDbSize(ns, appName, nid string) (int, error) {
	db, err := nocalhost_db.OpenApplicationLevelDB(ns, appName, nid, true)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return db.GetSize()
}
