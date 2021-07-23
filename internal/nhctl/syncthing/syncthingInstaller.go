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

package syncthing

import (
	"fmt"
	"io"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/syncthing/bin"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"nocalhost/pkg/nhctl/log"
)

var (
	downloadURLs = map[string]string{
		"linux": "https://codingcorp-generic.pkg.coding.net/nocalhost/syncthing/syncthing-linux-amd64.tar.gz" +
			"?version=%s",
		"arm64": "https://codingcorp-generic.pkg.coding.net/nocalhost/syncthing/syncthing-linux-arm64.tar.gz" +
			"?version=%s",
		"darwin": "https://codingcorp-generic.pkg.coding.net/nocalhost/syncthing/syncthing-macos-amd64.zip" +
			"?version=%s",
		"windows": "https://codingcorp-generic.pkg.coding.net/nocalhost/syncthing/syncthing-windows-amd64.zip" +
			"?version=%s",
	}

	downloadErrorTips = `Downloading syncthing error, please check your network!! you can try again, 

Or download syncthing by your own:

1, download syncthing bin from %s
2, cp bin into: %s
3, given syncthing Execution access permission, such as "chmod +x %s" in linux or macos.

Then, try reenter DevMode.
`
)

type SyncthingInstaller struct {
	BinPath  string
	Version  string
	CommitId string
}

func NewInstaller(binPath string, version string, commitId string) *SyncthingInstaller {
	return &SyncthingInstaller{
		BinPath:  binPath,
		Version:  version,
		CommitId: commitId,
	}
}

// the return val bool is only for test case, it shows that whether download again
func (s *SyncthingInstaller) InstallIfNeeded() (bool, error) {

	// first try to download the version matched Version
	// then try the version matched commit id
	downloadCandidate := s.needToDownloadByVersionAndCommitId()

	// download all the candidates version of syncthing
	// if download fail at all, use the local binPath's syncthing or else throw the error
	var maxIndex = len(downloadCandidate) - 1
	for i, version := range downloadCandidate {
		errTips, err := s.downloadSyncthing(version)
		if err != nil {
			if errTips != "" {
				fmt.Println()
				coloredoutput.Fail(errTips)
			}

			if maxIndex == i && !s.isInstalled() {
				return len(downloadCandidate) != 0, err
			}
		} else {
			// return while download success
			return len(downloadCandidate) != 0, nil
		}
	}

	return len(downloadCandidate) != 0, nil
}

func (s *SyncthingInstaller) downloadSyncthing(version string) (string, error) {
	t := time.NewTicker(1 * time.Second)
	var err error
	var errorTips string
	for i := 0; i < 3; i++ {
		errorTips, err = s.install(version)
		if err == nil {
			return "", nil
		}

		log.Error("Fail to copy syncthing, retry...")

		if i < 2 {
			<-t.C
		}
	}

	return errorTips, err
}

// Install installs syncthing
func (s *SyncthingInstaller) install(version string) (string, error) {
	var err error

	i := s.BinPath

	if FileExists(i) {
		if err := os.Remove(i); err != nil {
			log.Debugf("Failed to delete %s, will try to overwrite: %s", i, err)
		}
	}

	// fix if ~/.nh/nhctl/bin/syncthing not exist
	if err = os.MkdirAll(GetDir(i), 0700); err != nil {
		err = fmt.Errorf("failed mkdir for %s: %s", i, err)
		return "", err
	}

	if err = bin.CopyToBinPath(i); err != nil {
		err = fmt.Errorf("failed to copy file to: %s, err: %v", i, err)
	}

	fmt.Printf("copyed syncthing %s to %s\n", version, i)
	return "", nil
}

func (s *SyncthingInstaller) isInstalled() bool {
	_, err := os.Stat(s.BinPath)
	return !os.IsNotExist(err)
}

func (s *SyncthingInstaller) needToDownloadByVersionAndCommitId() []string {
	var installCandidate []string

	defer func() {
		if len(installCandidate) > 0 {
			log.Infof(
				"Need to download from following versions according to the sequence: %s",
				strings.Join(installCandidate, ", "),
			)
		}
	}()

	if s.Version != "" {
		if s.exec("-nocalhost") == s.Version {
			return installCandidate
		}
		installCandidate = append(installCandidate, s.Version)
	}

	if s.CommitId != "" {
		if s.exec("-nocalhost-commit-id") == s.CommitId {
			return installCandidate
		}
		installCandidate = append(installCandidate, s.CommitId)
	}

	return installCandidate
}

func (s *SyncthingInstaller) exec(flag string) string {
	cmdArgs := []string{
		flag,
	}
	output, err := exec.Command(s.BinPath, cmdArgs...).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSuffix(string(output), "\n")
}

func CopyFile(from, to string) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	toFile, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0700)
	if err != nil {
		return err
	}

	defer toFile.Close()

	_, err = io.Copy(toFile, fromFile)
	if err != nil {
		return err
	}

	return nil
}

func GetDir(path string) string {
	return path[:strings.LastIndex(path, string(os.PathSeparator))]
}

func FileExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		log.Debugf("Failed to check if %s exists: %s", name, err)
	}

	return true
}

func GetDownloadURL(os, arch, version string) (string, error) {
	src, ok := downloadURLs[os]
	if !ok {
		return "", fmt.Errorf("%s is not a supported platform", os)
	}

	if os == "linux" {
		switch arch {
		case "arm":
			return downloadURLs["arm"], nil
		case "arm64":
			return downloadURLs["arm64"], nil
		}
	}

	return fmt.Sprintf(src, version), nil
}

func getBinaryPathInDownload(dir, subdir string) string {
	return filepath.Join(dir, subdir, GetBinaryName())
}

func GetBinaryName() string {
	if runtime.GOOS == "windows" {
		return "syncthing.exe"
	}
	return "syncthing"
}
