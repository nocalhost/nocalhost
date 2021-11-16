//go:build windows
// +build windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package openvpn

import (
	"embed"
	"io/ioutil"
	"os"
	"os/exec"
)

//go:embed exe/tap-windows-9.21.2.exe
var fs embed.FS

//	driver download from https://build.openvpn.net/downloads/releases/
func Install() error {
	bytes, err := fs.ReadFile("exe/tap-windows-9.21.2.exe")
	if err != nil {
		return err
	}
	tempFile, err := ioutil.TempFile("", "*.exe")
	defer func() { _ = os.Remove(tempFile.Name()) }()
	if err != nil {
		return err
	}
	if _, err = tempFile.Write(bytes); err != nil {
		return err
	}
	_ = tempFile.Sync()
	_ = tempFile.Close()
	_ = os.Chmod(tempFile.Name(), 0700)
	cmd := exec.Command(tempFile.Name(), "/S")
	return cmd.Run()
}
