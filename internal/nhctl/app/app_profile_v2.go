/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"github.com/pkg/errors"
	"os"
)

func (a *Application) checkIfAppConfigIsV2() (bool, error) {
	_, err := os.Stat(a.GetConfigV2Path())
	if err == nil {
		return true, nil
	}

	if !os.IsNotExist(err) {
		return false, errors.Wrap(err, "")
	}
	return false, nil
}

func (a *Application) UpgradeAppConfigV1ToV2() error {
	err := ConvertConfigFileV1ToV2(a.GetConfigPath(), a.GetConfigV2Path())
	if err != nil {
		return err
	}
	return os.Rename(a.GetConfigPath(), a.GetConfigPath()+".bak")
}
