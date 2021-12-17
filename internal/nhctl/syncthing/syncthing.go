/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

// syncthing module: thanks to okteto given our inspired

package syncthing

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	_const "nocalhost/internal/nhctl/const"
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
	ignoredFileTemplate = template.Must(template.New("ignoredFileTemplate").Parse(local.IgnoredFileTemplate))
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
	IgnoredFIle      = ".nhignore"
	syncthingPidFile = "syncthing.pid"

	// Use to access syncthing API
	DefaultAPIKey = "nocalhost"

	// DefaultRemoteDeviceID remote syncthing ID
	DefaultRemoteDeviceID = "MDPJNTF-OSPJC65-LZNCQGD-3AWRUW6-BYJULSS-GOCA2TU-5DWWBNC-TKM4VQ5"
	localDeviceID         = "SJTYMUE-DI3REKX-JCLCRXU-F6UJHCG-XQGHAZJ-5O5D3JR-LALGSBC-TJ4I4QO"

	// DefaultFileWatcherDelay how much to wait before starting a sync after a file change
	DefaultFileWatcherDelay = 5

	// may result bug due to syncthing config changing
	DefaultFolderName = "nh-1"

	// ClusterPort is the port used by syncthing in the cluster
	ClusterPort = 22000

	// GUIPort is the port used by syncthing in the cluster for the http endpoint
	GUIPort = 8384

	EnableParseFromGitIgnore  = "#enableParseFromGitIgnore"
	DisableParseFromGitIgnore = "#disableParseFromGitIgnore"
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

	// resolve ignore/sync from pattern
	SyncedPattern  []string `yaml:"-"`
	IgnoredPattern []string `yaml:"-"`

	// resolve ignore/sync from gitignore
	EnableParseFromGitIgnore bool `yaml:"-"`
}

//IsSubPathFolder checks if a sync folder is a subpath of another sync folder
func IsSubPathFolder(path string, paths []string) (bool, error) {
	found := false
	for _, sync := range paths {
		rel, err := filepath.Rel(sync, path)
		if err != nil {
			log.Debugf("error making rel '%s' and '%s'", sync, path)
			return false, errors.Wrap(err, "")
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

func cleanupDaemon(pid int, wait bool) error {
	process, err := ps.FindProcess(pid)
	if process == nil && err == nil {
		return nil
	}

	if err != nil {
		return errors.Wrap(err, "")
	}

	if process.Executable() != GetBinaryName() {
		log.Infof("%d is not a syncthing process", pid)
		return nil
	}

	if err = terminate.Terminate(pid, wait); err == nil {
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
	bs, err := s.GetLocalConfigXML()
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(s.LocalHome, configFile), bs, 0700); err != nil {
		return fmt.Errorf("failed to write syncthing configuration file: %w", err)
	}

	return nil
}

// Generate s.LocalHome/.nhignore by file sync option
func (s *Syncthing) generateIgnoredFileConfig() (string, error) {
	var ignoreFilePath = filepath.Join(s.LocalHome, IgnoredFIle)
	var syncedPatternAdaption = make([]string, len(s.SyncedPattern))
	var enableParseFromGitIgnore = DisableParseFromGitIgnore

	for i, synced := range s.SyncedPattern {
		var afterAdapt = synced

		// previews version support such this syntax
		if synced == "." {
			afterAdapt = "**"
		}

		if strings.Index(synced, "./") == 0 {
			afterAdapt = synced[1:]
		}

		syncedPatternAdaption[i] = "!" + afterAdapt
	}

	var ignoredPatternAdaption = make([]string, len(s.IgnoredPattern))
	for i, ignored := range s.IgnoredPattern {
		var afterAdapt = ignored

		// previews version support such this syntax
		if ignored == "." {
			afterAdapt = "**"
		}

		if strings.Index(ignored, "./") == 0 {
			afterAdapt = ignored[1:]
		}

		ignoredPatternAdaption[i] = afterAdapt
	}

	if len(syncedPatternAdaption) == 0 {
		syncedPatternAdaption = []string{"!**"}
	}

	ignoredPattern := ""
	syncedPattern := ""

	if s.EnableParseFromGitIgnore {
		log.Infof("\n Enable parsing file's ignore/sync from git ignore\n")
		enableParseFromGitIgnore = EnableParseFromGitIgnore
	} else {
		ignoredPattern = strings.Join(ignoredPatternAdaption, "\n")
		syncedPattern = strings.Join(syncedPatternAdaption, "\n")

		log.Infof("IgnoredPattern: \n" + ignoredPattern)
		log.Infof("SyncedPattern: \n" + syncedPattern)
	}

	var values = map[string]string{
		"enableParseFromGitIgnore": enableParseFromGitIgnore,
		"ignoredPattern":           ignoredPattern,
		"syncedPattern":            syncedPattern,
	}

	buf := new(bytes.Buffer)
	if err := ignoredFileTemplate.Execute(buf, values); err != nil {
		return "", fmt.Errorf("failed to write .nhignore configuration template: %w", err)
	}

	if err := ioutil.WriteFile(ignoreFilePath, buf.Bytes(), _const.DefaultNewFilePermission); err != nil {
		return "", fmt.Errorf("failed to generate .nhignore configuration: %w", err)
	}

	return ignoreFilePath, nil
}

// Run local syncthing server
func (s *Syncthing) Run(ctx context.Context) error {
	if err := s.initConfig(); err != nil {
		log.Warnf("fail to init syncthing: %s", err.Error())
		return err
	}

	ignoreFilePath, err := s.generateIgnoredFileConfig()

	if err != nil {
		log.Warnf("fail to generate ignore file config: %s", err.Error())
		ignoreFilePath = ""
	}

	pidPath := filepath.Join(s.LocalHome, syncthingPidFile)

	cmdArgs := []string{
		"serve",
		"--home", s.LocalHome,
		"--no-browser",
		"--verbose",
		"--logfile", s.LogPath,
		"--log-max-old-files=0",
		"--ignore-file-path=" + ignoreFilePath,
	}

	log.Debug(cmdArgs)
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

// Stop syncthing background process
func Stop(pid int, force bool) error {
	if err := cleanupDaemon(pid, force); err != nil {
		return err
	}
	return nil
}

func (s *Syncthing) GetRemoteConfigXML() ([]byte, error) {
	configTemplate := template.Must(template.New("syncthingConfig").Parse(local.RemoteSyncConfigXML))
	buf := new(bytes.Buffer)
	if err := configTemplate.Execute(buf, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Syncthing) GetLocalConfigXML() ([]byte, error) {
	configTemplate := template.Must(template.New("syncthingConfig").Parse(local.LocalSyncConfigXML))
	buf := new(bytes.Buffer)
	if err := configTemplate.Execute(buf, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
