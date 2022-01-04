/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_common

import (
	"context"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
)

const (
	DefaultDaemonPort  = 30123
	SudoDaemonPort     = 30124
	DaemonHttpPort     = 30125
	SudoDaemonHttpPort = 30126
)

var (
	Version  = "1.0"
	CommitId = ""
)

type DaemonServerInfo struct {
	Version   string
	CommitId  string
	NhctlPath string
	Upgrading bool
}

type CheckClusterStatus struct {
	Available bool
	Info      string
	//FailedCount int
}

type PortForwardProfile struct {
	Cancel     context.CancelFunc `json:"-"` // For canceling a port forward
	StopCh     chan error         `json:"-"`
	NameSpace  string             `json:"nameSpace"`
	AppName    string             `json:"appName"`
	SvcName    string             `json:"svcName"`
	SvcType    string             `json:"svcType"`
	Role       string             `json:"role"`
	LocalPort  int                `json:"localPort"`
	RemotePort int                `json:"remotePort"`
}

type DaemonServerStatusResponse struct {
	PortForwardList []*PortForwardProfile `json:"portForwardList"`
}

// StartDaemonServerBySubProcess
// Start daemon server from client
func StartDaemonServerBySubProcess(isSudoUser bool) error {
	var (
		nhctlPath string
		err       error
	)
	//if utils.IsWindows() {
	//	if nhctlPath, err = CopyNhctlBinaryToTmpDir(os.Args[0]); err != nil {
	//		return err
	//	}
	//} else {
	if nhctlPath, err = utils.GetNhctlPath(); err != nil {
		return err
	}
	//}
	daemonArgs := []string{nhctlPath, "daemon", "start"}
	if isSudoUser {
		daemonArgs = append(daemonArgs, "--sudo", "true")
	}
	return daemon.RunSubProcess(daemonArgs, nil, false)
}

// CopyNhctlBinaryToTmpDir
// Copy nhctl binary to a tmpDir and return the path of nhctl in tmpDir
func CopyNhctlBinaryToTmpDir(nhctlPath string) (string, error) {
	daemonDir, err := ioutil.TempDir("", "nhctl-daemon")
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	// cp nhctl to daemonDir
	if err = utils.CopyFile(nhctlPath, filepath.Join(daemonDir, utils.GetNhctlBinName())); err != nil {
		return "", errors.Wrap(err, "")
	}
	return filepath.Join(daemonDir, utils.GetNhctlBinName()), nil
}
