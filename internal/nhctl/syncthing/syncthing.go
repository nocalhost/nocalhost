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

// syncthing module: thanks to okteto given our inspired

package syncthing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	ps "github.com/mitchellh/go-ps"

	"nocalhost/internal/nhctl/syncthing/local"
	"nocalhost/internal/nhctl/syncthing/terminate"
	"nocalhost/pkg/nhctl/log"
)

var (
	configTemplate = template.Must(template.New("syncthingConfig").Parse(local.ConfigXML))
)

const (
	SyncSecretName   = "nocalhost-syncthing-secret"
	syncthing        = "syncthing"
	RemoteHome       = "/var/syncthing/config"
	Bind             = "127.0.0.1"
	Nocalhost        = "nocalhost"
	certFile         = "cert.pem"
	keyFile          = "key.pem"
	configFile       = "config.xml"
	LogFile          = "syncthing.log"
	syncthingPidFile = "syncthing.pid"
	DefaultSyncMode  = "sendreceive" // default sync mode
	SendOnlySyncMode = "sendonly"    // default sync mode
	// DefaultRemoteDeviceID remote syncthing ID
	DefaultRemoteDeviceID = "MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5"
	localDeviceID         = "SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO"

	// DefaultFileWatcherDelay how much to wait before starting a sync after a file change
	DefaultFileWatcherDelay = 5

	// ClusterPort is the port used by syncthing in the cluster
	ClusterPort = 22000

	// GUIPort is the port used by syncthing in the cluster for the http endpoint
	GUIPort = 8384
)

//Folder represents a sync folder
type Folder struct {
	Name         string `yaml:"name"`
	LocalPath    string `yaml:"localPath"`
	RemotePath   string `yaml:"remotePath"`
	Retries      int    `yaml:"-"`
	SentStIgnore bool   `yaml:"-"`
}

//Ignores represents the .stignore file
type Ignores struct {
	Ignore []string `json:"ignore"`
}

type Syncthing struct {
	APIKey                   string       `yaml:"apikey"`
	GUIPassword              string       `yaml:"password"`
	GUIPasswordHash          string       `yaml:"-"`
	BinPath                  string       `yaml:"-"`
	Client                   *http.Client `yaml:"-"`
	cmd                      *exec.Cmd    `yaml:"-"`
	Folders                  []*Folder    `yaml:"folders"`
	FileWatcherDelay         int          `yaml:"-"`
	ForceSendOnly            bool         `yaml:"-"`
	GUIAddress               string       `yaml:"local"`
	LocalHome                string       `yaml:"-"`
	RemoteHome               string       `yaml:"-"`
	LogPath                  string       `yaml:"-"`
	ListenAddress            string       `yaml:"-"`
	RemoteAddress            string       `yaml:"-"`
	RemoteDeviceID           string       `yaml:"-"`
	RemoteGUIAddress         string       `yaml:"remote"`
	RemoteGUIPort            int          `yaml:"-"`
	RemotePort               int          `yaml:"-"`
	LocalGUIPort             int          `yaml:"-"`
	LocalPort                int          `yaml:"-"`
	Type                     string       `yaml:"-"`
	IgnoreDelete             bool         `yaml:"-"`
	PortForwardBackGroundPid int          `yaml:"-"`
	SyncthingBackGroundPid   int          `yaml:"-"`
	pid                      int          `yaml:"-"`
	RescanInterval           string       `yaml:"-"`
}

// move to nocalhost/internal/nhctl/app
// New constructs a new Syncthing.
//func New(dev *napp.Application, deployment string, devStartOptions *napp.DevStartOptions, fileSyncOptions *napp.FileSyncOptions) (*Syncthing, error) {
//	var err error
//
//	// 2. from file-sync it should direct use .profile.yaml dev-start port
//	// WorkDir will be null
//	remotePort := fileSyncOptions.RemoteSyncthingPort
//	remoteGUIPort := fileSyncOptions.RemoteSyncthingGUIPort
//	localListenPort := fileSyncOptions.LocalSyncthingPort
//	localGuiPort := fileSyncOptions.LocalSyncthingGUIPort
//
//	// 1. from dev-start should take some local available port, WorkDir will be not null
//	if devStartOptions.WorkDir != "" {
//		remotePort, err = ports.GetAvailablePort()
//		if err != nil {
//			return nil, err
//		}
//
//		remoteGUIPort, err = ports.GetAvailablePort()
//		if err != nil {
//			return nil, err
//		}
//
//		localGuiPort, err = ports.GetAvailablePort()
//		if err != nil {
//			return nil, err
//		}
//
//		localListenPort, err = ports.GetAvailablePort()
//		if err != nil {
//			return nil, err
//		}
//	}
//	hash, err := bcrypt.GenerateFromPassword([]byte(Nocalhost), 0)
//	if err != nil {
//		log.Debugf("couldn't hash the password %s", err)
//		hash = []byte("")
//	}
//	sendMode := DefaultSyncMode
//	if !fileSyncOptions.SyncDouble {
//		sendMode = SendOnlySyncMode
//	}
//	s := &Syncthing{
//		APIKey:           "nocalhost",
//		GUIPassword:      "nocalhost",
//		GUIPasswordHash:  string(hash),
//		BinPath:          filepath.Join(dev.GetSyncThingBinDir(), GetBinaryName()),
//		Client:           NewAPIClient(),
//		FileWatcherDelay: DefaultFileWatcherDelay,
//		GUIAddress:       fmt.Sprintf("%s:%d", Bind, localGuiPort),
//		// TODO BE CAREFUL ResourcePath if is not application path, this is Local syncthing HOME PATH, use for cert and config.xml
//		// it's `~/.nhctl/application/bookinfo/syncthing`
//		LocalHome:        filepath.Join(dev.GetHomeDir(), syncthing, deployment),
//		RemoteHome:       RemoteHome,
//		LogPath:          filepath.Join(dev.GetHomeDir(), syncthing, deployment, LogFile),
//		RemoteAddress:    fmt.Sprintf("%s:%d", Bind, remotePort),
//		RemoteDeviceID:   DefaultRemoteDeviceID,
//		RemoteGUIAddress: fmt.Sprintf("%s:%d", Bind, remoteGUIPort),
//		RemoteGUIPort:    remoteGUIPort,
//		RemotePort:       remotePort,
//		LocalGUIPort:     localGuiPort,
//		LocalPort:        localListenPort,
//		ListenAddress:    fmt.Sprintf("%s:%d", Bind, localListenPort),
//		Type:             sendMode, // sendonly mode
//		IgnoreDelete:     true,
//		Folders:          []*Folder{},
//		RescanInterval:   "300",
//	}
//
//	// when create syncthing sidecar it need to know how many directory it should sync
//	index := 1
//	for _, sync := range devStartOptions.LocalSyncDir {
//		result, err := IsSubPathFolder(sync, devStartOptions.LocalSyncDir)
//		// TODO consider continue handle next dir
//		if err != nil {
//			return nil, err
//		}
//		if !result {
//			s.Folders = append(
//				s.Folders,
//				&Folder{
//					Name:       strconv.Itoa(index),
//					LocalPath:  sync,
//					RemotePath: devStartOptions.WorkDir,
//				},
//			)
//			index++
//		}
//	}
//
//	return s, nil
//}

//IsSubPathFolder checks if a sync folder is a subpath of another sync folder
func IsSubPathFolder(path string, paths []string) (bool, error) {
	found := false
	for _, sync := range paths {
		rel, err := filepath.Rel(sync, path)
		if err != nil {
			log.Debugf("error making rel '%s' and '%s'", sync, path)
			return false, err
		}
		if strings.HasPrefix(rel, "..") {
			continue
		}
		found = true
		if rel != "." {
			return true, nil
		}
	}
	if found {
		return false, nil
	}
	return false, errors.New("not found")
}

func (s *Syncthing) cleanupDaemon(pid int, wait bool, typeName string) error {
	process, err := ps.FindProcess(pid)
	if process == nil && err == nil {
		return nil
	}

	if err != nil {
		log.Fatalf("error when looking up the process: %s", err)
		return err
	}

	if typeName == syncthing {
		if process.Executable() != GetBinaryName() {
			return nil
		}
	}

	err = terminate.Terminate(pid, wait, typeName)
	if err == nil {
		log.Debugf("terminated syncthing with pid %d", pid)
	}

	return err
}

func (s *Syncthing) initConfig() error {
	if err := os.MkdirAll(s.LocalHome, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %s", s.LocalHome, err)
	}

	if err := s.UpdateConfig(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(s.LocalHome, certFile), local.Cert, 0700); err != nil {
		return fmt.Errorf("failed to write syncthing certificate: %w", err)
	}

	if err := ioutil.WriteFile(filepath.Join(s.LocalHome, keyFile), local.Key, 0700); err != nil {
		return fmt.Errorf("failed to write syncthing key: %w", err)
	}

	return nil
}

// UpdateConfig updates the syncthing config file
func (s *Syncthing) UpdateConfig() error {
	buf := new(bytes.Buffer)
	if err := configTemplate.Execute(buf, s); err != nil {
		return fmt.Errorf("failed to write syncthing configuration template: %w", err)
	}

	if err := ioutil.WriteFile(filepath.Join(s.LocalHome, configFile), buf.Bytes(), 0700); err != nil {
		return fmt.Errorf("failed to write syncthing configuration file: %w", err)
	}

	return nil
}

// Run local syncthing server
func (s *Syncthing) Run(ctx context.Context) error {
	if err := s.initConfig(); err != nil {
		log.Warnf("fail to init syncthing: %s", err.Error())
		return err
	}

	//val := os.Getenv("STNOUPGRADE")
	//if val == "1" { // child process
	//	return nil
	//}

	pidPath := filepath.Join(s.LocalHome, syncthingPidFile)

	cmdArgs := []string{
		"-home", s.LocalHome,
		"-no-browser",
		"-verbose",
		"-logfile", s.LogPath,
		"-log-max-old-files=0",
	}
	s.cmd = exec.Command(s.BinPath, cmdArgs...) //nolint: gas, gosec
	s.cmd.Env = append(os.Environ(), "STNOUPGRADE=1")

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start syncthing: %w", err)
	}

	if s.cmd.Process == nil {
		return nil
	}

	if err := ioutil.WriteFile(pidPath, []byte(strconv.Itoa(s.cmd.Process.Pid)), 0600); err != nil {
		return fmt.Errorf("failed to write syncthing pid file: %w", err)
	}

	s.pid = s.cmd.Process.Pid

	log.Debugf("local syncthing pid-%d running", s.pid)
	return nil
}

// Stop syncthing background process or port-forward progress, and clean up pid files.
func (s *Syncthing) Stop(pid int, pidFilePath string, typeName string, force bool) error {
	if err := s.cleanupDaemon(pid, force, typeName); err != nil {
		return err
	}

	if err := os.Remove(pidFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		fmt.Printf("failed to delete pidfile %d: %s", pid, err)
	}
	return nil
}

func getPID(pidPath string) (int, error) {
	if _, err := os.Stat(pidPath); err != nil {
		return 0, err
	}

	content, err := ioutil.ReadFile(pidPath) // nolint: gosec
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(content))
}
