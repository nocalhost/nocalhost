/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import nocalhostdb "nocalhost/internal/nhctl/nocalhost/db"

func UpdateKey(ns, app, nid string, key string, value string) error {
	db, err := nocalhostdb.OpenApplicationLevelDB(ns, app, nid, false)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Put([]byte(key), []byte(value))
}
