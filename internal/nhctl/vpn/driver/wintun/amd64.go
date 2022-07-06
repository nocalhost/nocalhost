//go:build windows && amd64
// +build windows,amd64

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package wintun

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:embed bin/amd64/wintun.dll
var wintunFs embed.FS

func InstallWintunDriver() error {
	bytes, err := wintunFs.ReadFile("bin/amd64/wintun.dll")
	if err != nil {
		return err
	}
	currentFile, err := os.Executable()
	if err != nil {
		return err
	}
	filename := filepath.Join(filepath.Dir(currentFile), "wintun.dll")
	_ = os.Remove(filename)
	err = ioutil.WriteFile(filename, bytes, 644)
	return err
}
