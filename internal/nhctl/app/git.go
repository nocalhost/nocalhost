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

package app

import (
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/utils"
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
			_, err = tools.ExecCommand(nil, true, true, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, destPath)
		} else {
			_, err = tools.ExecCommand(nil, true, true, "git", "clone", "--depth", "1", gitUrl, destPath)
		}
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
	}
	return nil

}

func (a *Application) downloadResourcesFromGit(gitUrl, gitRef, destPath string) error {
	return cloneFromGit(gitUrl, gitRef, destPath)
}

func (a *Application) copyUpgradeResourcesFromLocalDir(localDir string) error {

	_, err := os.Stat(a.getUpgradeGitDir())
	if err == nil {
		err = os.RemoveAll(a.getUpgradeGitDir())
		if err != nil {
			return errors.Wrap(err, "")
		}
	}

	err = os.Mkdir(a.getUpgradeGitDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(utils.CopyDir(localDir, a.getUpgradeGitDir()), "")
}

func (a *Application) downloadUpgradeResourcesFromGit(gitUrl string, gitRef string) error {
	_, err := os.Stat(a.getUpgradeGitDir())
	if err == nil {
		err = os.RemoveAll(a.getUpgradeGitDir())
		if err != nil {
			return errors.Wrap(err, "")
		}
	}

	err = os.Mkdir(a.getUpgradeGitDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return cloneFromGit(gitUrl, gitRef, a.getUpgradeGitDir())
}

func (a *Application) saveUpgradeResources() error {
	return moveDir(a.getUpgradeGitDir(), a.ResourceTmpDir)
}

// Move srcDir to destDir
// If destDir already exists, deleting it before move srcDir
func moveDir(srcDir string, destDir string) error {
	_, err := os.Stat(destDir)
	if err == nil {
		err = os.RemoveAll(destDir)
		if err != nil {
			return errors.Wrap(err, "")
		}
	}
	return errors.Wrap(os.Rename(srcDir, destDir), "")
}
