/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package syncthing

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/fs"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/syncthing/bin"
	"nocalhost/internal/nhctl/utils"
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
		"linux-amd64": "https://nocalhost-generic.pkg.coding.net/nocalhost/syncthing/syncthing-linux-amd64.tar.gz" +
			"?version=%s",
		"darwin-arm64": "https://nocalhost-generic.pkg.coding.net/nocalhost/syncthing/syncthing-macos-arm64.tar.gz" +
			"?version=%s",
		"darwin-amd64": "https://nocalhost-generic.pkg.coding.net/nocalhost/syncthing/syncthing-macos-amd64.zip" +
			"?version=%s",
		"windows-amd64": "https://nocalhost-generic.pkg.coding.net/nocalhost/syncthing/syncthing-windows-amd64.zip" +
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
	syncthingCandidate, needCopy := s.getCompatibilityByVersionAndCommitId()
	if !needCopy {
		return false, nil
	}

	// download all the candidates version of syncthing
	// if download fail at all, use the local binPath's syncthing or else throw the error
	var maxIndex = len(syncthingCandidate) - 1
	for i, version := range syncthingCandidate {
		errTips, err := s.downloadSyncthing(version)
		if err != nil {
			if errTips != "" {
				fmt.Println()
				coloredoutput.Fail(errTips)
			}

			if maxIndex == i && !s.isInstalled() {
				return len(syncthingCandidate) != 0, err
			}
		} else {
			// return while download success
			return len(syncthingCandidate) != 0, nil
		}
	}

	return len(syncthingCandidate) != 0, nil
}

func (s *SyncthingInstaller) downloadSyncthing(version string) (string, error) {
	t := time.NewTicker(1 * time.Second)
	var err error
	var errorTips string
	for i := 0; i < 3; i++ {
		//p := &utils.ProgressBar{}
		//d := 40 * time.Second
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

	_ = filepath.WalkDir(filepath.Dir(i), func(path string, d fs.DirEntry, err error) error {
		_ = os.Remove(path)
		return nil
	})

	if FileExists(i) {
		log.Debugf("Failed to delete %s, will try to overwrite: %s", i, err)
		if err = os.Rename(i, filepath.Join(filepath.Dir(i), uuid.New().String()+filepath.Base(i))); err != nil {
			log.Debugf(fmt.Sprintf("Can't rename file: %s --> %s", i, uuid.New().String()+filepath.Base(i)))
			if utils.IsWindows() {
				return "", nil
			}
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

	log.Infof("copyed syncthing %s to %s\n", version, i)
	return "", nil
}

func (s *SyncthingInstaller) isInstalled() bool {
	_, err := os.Stat(s.BinPath)
	return !os.IsNotExist(err)
}

func (s *SyncthingInstaller) getCompatibilityByVersionAndCommitId() ([]string, bool) {
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
		if s.exec("serve", "--nocalhost") == s.Version {
			return installCandidate, false
		}
		installCandidate = append(installCandidate, s.Version)
	}

	if s.CommitId != "" {
		if s.exec("serve", "--nocalhost-commit-id") == s.CommitId {
			return installCandidate, false
		}
		installCandidate = append(installCandidate, s.CommitId)
	}

	return installCandidate, true
}

func (s *SyncthingInstaller) exec(flags ...string) string {

	output, err := exec.Command(s.BinPath, flags...).Output()
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
	osArch := fmt.Sprintf("%s-%s", os, arch)

	src, ok := downloadURLs[osArch]
	if !ok {
		return "", fmt.Errorf("%s is not supported temporary!! you can contact us to support this version. ", osArch)
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
