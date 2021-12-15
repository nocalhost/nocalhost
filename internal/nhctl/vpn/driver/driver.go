/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package driver

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/vpn/driver/wintun"
	"os"
	"path/filepath"
)

func InstallWireGuardTunDriver() {
	if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return err != nil
	}, func() error {
		return wintun.InstallWintunDriver()
	}); err != nil {
		log.Warn(err)
	}
}

func UninstallWireGuardTunDriver() error {
	currentFile, err := os.Executable()
	if err != nil {
		return err
	}
	filename := filepath.Join(filepath.Dir(currentFile), "wintun.dll")
	return os.Remove(filename)
}
