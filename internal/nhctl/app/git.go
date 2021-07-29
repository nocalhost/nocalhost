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

package app

import (
	"github.com/pkg/errors"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"strings"
)

func cloneFromGit(gitUrl string, gitRef string, destPath string) error {

	var (
		err        error
		gitDirName string
	)

	if strings.HasPrefix(gitUrl, "https") || strings.HasPrefix(gitUrl, "git") || strings.HasPrefix(gitUrl, "http") {
		if strings.HasSuffix(gitUrl, ".git") {
			gitDirName = gitUrl[:len(gitUrl)-4]
		} else {
			gitDirName = gitUrl
		}
		strs := strings.Split(gitDirName, "/")
		gitDirName = strs[len(strs)-1] // todo : for default application name
		if len(gitRef) > 0 {
			_, err = tools.ExecCommand(
				nil, true, true, false, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, destPath,
				"--config", "core.sshCommand=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no",
			)
		} else {
			_, err = tools.ExecCommand(
				nil, true, true, false, "git", "clone", "--depth", "1", gitUrl, destPath,
				"--config", "core.sshCommand=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no",
			)
		}
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
	}
	return nil

}

func downloadResourcesFromGit(gitUrl, gitRef, destPath string) error {
	return cloneFromGit(gitUrl, gitRef, destPath)
}

// remove srcDir to destDir
// If destDir already exists, deleting it before move srcDir
func removeDir(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		return errors.Wrap(os.RemoveAll(dir), "")
	}
	return nil
}
