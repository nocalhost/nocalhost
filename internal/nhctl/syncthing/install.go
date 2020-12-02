/*
Copyright 2020 The Nocalhost Authors.
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

package syncthing

import (
	"fmt"
	"github.com/Masterminds/semver"
	getter "github.com/hashicorp/go-getter"
	"io"
	"io/ioutil"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const syncthingVersion = "1.11.1"

var (
	downloadURLs = map[string]string{
		"linux":   fmt.Sprintf("https://github.com/syncthing/syncthing/releases/download/v%[1]s/syncthing-linux-amd64-v%[1]s.tar.gz", syncthingVersion),
		"arm64":   fmt.Sprintf("https://github.com/syncthing/syncthing/releases/download/v%[1]s/syncthing-linux-arm64-v%[1]s.tar.gz", syncthingVersion),
		"darwin":  fmt.Sprintf("https://github.com/syncthing/syncthing/releases/download/v%[1]s/syncthing-macos-amd64-v%[1]s.zip", syncthingVersion),
		"windows": fmt.Sprintf("https://github.com/syncthing/syncthing/releases/download/v%[1]s/syncthing-windows-amd64-v%[1]s.zip", syncthingVersion),
	}

	minimumVersion = semver.MustParse(syncthingVersion)
	versionRegex   = regexp.MustCompile(`syncthing v(\d+\.\d+\.\d+) .*`)
)

func (s *Syncthing) DownloadSyncthing() error {
	t := time.NewTicker(1 * time.Second)
	var err error
	for i := 0; i < 3; i++ {
		p := &utils.ProgressBar{}
		err = s.Install(p)
		if err == nil {
			return nil
		}

		if i < 2 {
			log.Fatalf("failed to download syncthing, retrying: %s", err)
			<-t.C
		}
	}

	return err
}

// Install installs syncthing
func (s *Syncthing) Install(p getter.ProgressTracker) error {
	fmt.Printf("installing syncthing for %s/%s", runtime.GOOS, runtime.GOARCH)
	downloadURL, err := GetDownloadURL(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	opts := []getter.ClientOption{}
	if p != nil {
		opts = []getter.ClientOption{getter.WithProgress(p)}
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp download dir")
	}

	client := &getter.Client{
		Src:     downloadURL,
		Dst:     dir,
		Mode:    getter.ClientModeDir,
		Options: opts,
	}

	defer os.RemoveAll(dir)

	if err := client.Get(); err != nil {
		return fmt.Errorf("failed to download syncthing from %s: %s", client.Src, err)
	}

	i := s.binPath
	b := getBinaryPathInDownload(dir, downloadURL)

	if _, err := os.Stat(b); err != nil {
		return fmt.Errorf("%s didn't include the syncthing binary: %s", downloadURL, err)
	}

	if err := os.Chmod(b, 0700); err != nil {
		return fmt.Errorf("failed to set permissions to %s: %s", b, err)
	}

	if FileExists(i) {
		if err := os.Remove(i); err != nil {
			log.Debugf("failed to delete %s, will try to overwrite: %s", i, err)
		}
	}

	if err := CopyFile(b, i); err != nil {
		return fmt.Errorf("failed to write %s: %s", i, err)
	}

	fmt.Printf("downloaded syncthing %s to %s\n", syncthingVersion, i)
	return nil
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

func (s *Syncthing) IsInstalled() bool {
	_, err := os.Stat(path.Join(s.binPath, getBinaryName()))
	return !os.IsNotExist(err)
}

func GetDownloadURL(os, arch string) (string, error) {
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

	return src, nil
}

func getBinaryPathInDownload(dir, url string) string {
	_, f := path.Split(url)
	f = strings.TrimSuffix(f, ".tar.gz")
	f = strings.TrimSuffix(f, ".zip")
	return path.Join(dir, f, getBinaryName())
}

func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return "syncthing.exe"
	}
	return "syncthing"
}
