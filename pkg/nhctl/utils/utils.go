/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"fmt"
	"io"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/terminate"
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

func KillSyncthingProcessOnWindows(namespace, appname, servicename string, svcType base.SvcType) {
	cmd := exec.Command("wmic", "process", "get", "processid", ",", "commandline")
	s, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	var dirPath string
	if svcType == base.Deployment {
		dirPath = filepath.Join("nhctl", "ns", namespace, appname, nocalhost.DefaultBinSyncThingDirName, servicename)
	} else {
		dirPath = filepath.Join("nhctl", "ns", namespace, appname, nocalhost.DefaultBinSyncThingDirName, string(svcType)+"-"+servicename)
	}
	for _, item := range strings.Split(string(s), "\n") {
		if strings.Contains(item, dirPath) {
			for _, segment := range strings.Split(item, " ") {
				if pid, err := strconv.Atoi(segment); err == nil {
					_ = terminate.Terminate(pid, false)
				}
			}
		}
	}
}

func KillSyncthingProcessOnUnix(namespace, appname, servicename string, svcType base.SvcType) {
	var dirPath string
	if svcType == base.Deployment {
		dirPath = filepath.Join("nhctl", "ns", namespace, appname, nocalhost.DefaultBinSyncThingDirName, servicename)
	} else {
		dirPath = filepath.Join("nhctl", "ns", namespace, appname, nocalhost.DefaultBinSyncThingDirName, string(svcType)+"-"+servicename)
	}
	command := exec.Command("sh", "-c", "ps -ef | grep "+dirPath+"  | awk -F ' ' '{print $2}' | xargs kill")
	_ = command.Run()
}
