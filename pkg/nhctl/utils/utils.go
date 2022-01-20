/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package utils

import (
	"fmt"
	"io"
	"nocalhost/internal/nhctl/syncthing/terminate"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
)

func Mush(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		panic(err)
	}
}

type ProgressBar struct {
	lock sync.Mutex
}

func (cpb *ProgressBar) TrackProgress(src string, currentSize, totalSize int64, stream io.ReadCloser) io.ReadCloser {
	cpb.lock.Lock()
	defer cpb.lock.Unlock()

	newPb := pb.New64(totalSize)
	newPb.Set("prefix", fmt.Sprintf("%s ", filepath.Base(src)))
	newPb.SetCurrent(currentSize)
	newPb.Start()
	reader := newPb.NewProxyReader(stream)

	return &readCloser{
		Reader: reader,
		close: func() error {
			cpb.lock.Lock()
			defer cpb.lock.Unlock()

			newPb.Finish()
			return nil
		},
	}
}

type readCloser struct {
	io.Reader
	close func() error
}

func (c *readCloser) Close() error { return c.close() }

func RenderProgressBar(prefix string, current, scalingFactor float64) string {
	var sb strings.Builder
	_, _ = sb.WriteString(prefix)
	_, _ = sb.WriteString("[")

	scaledMax := int(100 * scalingFactor)
	scaledCurrent := int(current * scalingFactor)

	switch {
	case scaledCurrent == 0:
		_, _ = sb.WriteString(strings.Repeat("_", scaledMax))
	case scaledCurrent >= scaledMax:
		_, _ = sb.WriteString(strings.Repeat("-", scaledMax))
	default:
		_, _ = sb.WriteString(strings.Repeat("-", scaledCurrent-1))
		_, _ = sb.WriteString(">")
		_, _ = sb.WriteString(strings.Repeat("_", scaledMax-scaledCurrent))
	}

	_, _ = sb.WriteString("]")
	_, _ = sb.WriteString(fmt.Sprintf(" %3v%%", int(current)))
	return sb.String()
}

func KillSyncthingProcess(keyword string) {
	if utils.IsWindows() {
		KillSyncthingProcessOnWindows(keyword)
	} else {
		KillSyncthingProcessOnUnix(keyword)
	}
}

func KillSyncthingProcessOnWindows(keyword string) {
	cmd := exec.Command("wmic", "process", "get", "processid", ",", "commandline")
	s, err := cmd.CombinedOutput()
	log.Logf("execute wmic command, keyword: %s, error: %v", keyword, err)
	if err != nil {
		return
	}
	for _, item := range strings.Split(string(s), "\n") {
		if strings.Contains(item, keyword) {
			for _, segment := range strings.Split(item, " ") {
				if pid, err1 := strconv.Atoi(segment); err1 == nil {
					err = terminate.Terminate(pid, false)
					log.Logf("terminate syncthing pid: %v, error: %v", pid, err)
				}
			}
		}
	}
}

func KillSyncthingProcessOnUnix(keyword string) {
	command := exec.Command("sh", "-c", "ps -ef | grep "+keyword+"  | awk -F ' ' '{print $2}'")
	output, err := command.Output()
	ids := strings.Split(string(output), "\n")
	for _, id := range ids {
		if i, err := strconv.Atoi(id); err == nil {
			_ = terminate.Terminate(i, false)
		}
	}
	log.Logf("kill syncthing process, keyword: %s, command: %v, err: %v", keyword, command.Args, err)
}
