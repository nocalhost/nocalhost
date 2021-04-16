/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"time"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset application",
	Long:  `reset application`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if err := Prepare(); err != nil {
			log.FatalE(err, "")
		}

		if len(args) > 0 {
			applicationName := args[0]
			if applicationName != "" {
				if applicationName == app.DefaultNocalhostApplication {
					log.Error(app.DefaultNocalhostApplicationOperateErr)
					return
				}
				resetApplication(applicationName)
				return
			}
		}

		// Reset all applications under specify namespace
		appMap, err := nocalhost.GetNsAndApplicationInfo()
		mustI(err, "Failed to get applications")
		for ns, appList := range appMap {
			if ns != nameSpace {
				continue
			}
			for _, appName := range appList {
				resetApplication(appName)
			}
		}
		// remove ns dir
		time.Sleep(1 * time.Second)
		if nameSpace != "" {
			nsDir := filepath.Join(nocalhost_path.GetNhctlNameSpaceDir(), nameSpace)
			log.Infof("Removing ns dir : %s", nsDir)
			must(errors.Wrap(os.RemoveAll(nsDir), ""))
		}
	},
}

func resetApplication(applicationName string) {
	var err error
	initApp(applicationName)
	// Stop BackGroup Process
	appProfile, _ := nocalhostApp.GetProfile()
	for _, profile := range appProfile.SvcProfile {
		if profile.Developing {
			if err = nocalhostApp.StopSyncAndPortForwardProcess(profile.ActualName, true); err != nil {
				log.WarnE(err, "")
			}
		} else if len(profile.DevPortForwardList) > 0 {
			if err = nocalhostApp.StopAllPortForward(profile.ActualName); err != nil {
				log.WarnE(err, "")
			}
		}
	}

	// Remove files
	time.Sleep(1 * time.Second)
	if err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace); err != nil {
		log.WarnE(err, "")
	} else {
		log.Info("Files have been clean up")
	}
	log.Infof("Application %s has been reset.\n", applicationName)
}
