/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"time"
)

func cronJobForUpdatingHub() {
	for {
		hubDir := nocalhost_path.GetNocalhostHubDir()
		_, err := os.Stat(hubDir)
		if err != nil {
			if os.IsNotExist(err) {
				// git clone
				log.Log("Cloning nocalhost hub...")
				gitCloneParams := []string{"clone", "--depth", "1", "https://github.com/nocalhost/nocalhost-hub.git", hubDir}
				out, err := tools.ExecCommand(context.Background(), false, false, false, "git", gitCloneParams...)
				if err != nil {
					log.ErrorE(err, out)
				}
			} else {
				log.WarnE(errors.Wrap(err, ""), "Failed to stat nocalhost hub dir")
			}
		} else {
			// git pull
			log.Log("Pulling nocalhost hub...")
			gitPullParams := []string{"-C", hubDir, "pull"}
			out, err := tools.ExecCommand(context.Background(), false, false, false, "git", gitPullParams...)
			if err != nil {
				log.ErrorE(err, out)
			}
		}
		<-time.Tick(time.Minute * 2)
	}
}
