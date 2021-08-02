/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
