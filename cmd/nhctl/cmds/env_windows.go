// +build windows

/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func relError(file, path string) error {
	return fmt.Errorf("%s resolves to executable in current directory (.%c%s)", file, filepath.Separator, path)
}

// LookPath searches for an executable named file in the directories
// named by the PATH environment variable. If file contains a slash,
// it is tried directly and the PATH is not consulted. The result will be
// an absolute path.
//
// LookPath differs from exec.LookPath in its handling of PATH lookups,
// which are used for file names without slashes. If exec.LookPath's
// PATH lookup would have returned an executable from the current directory,
// LookPath instead returns an error.
func LookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if err != nil {
		return "", err
	}
	if filepath.Base(file) == file && !filepath.IsAbs(path) {
		return "", relError(file, path)
	}
	return path, nil
}

//initializeGW todo
func initializeGW() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\GitForWindows`, registry.QUERY_VALUE)
	if err != nil {
		if k, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\GitForWindows`, registry.QUERY_VALUE); err != nil {
			return "", err
		}
	}
	defer k.Close()
	installPath, _, err := k.GetStringValue("InstallPath")
	if err != nil {
		return "", err
	}
	installPath = filepath.Clean(installPath)
	git := filepath.Join(installPath, "cmd\\git.exe")
	if _, err := os.Stat(git); err != nil {
		return "", err
	}
	return filepath.Join(installPath, "cmd"), nil
}

func initializeEnv() error {
	var gitbin string
	if _, err := LookPath("git"); err != nil {
		if gitbin, err = initializeGW(); err != nil {
			return fmt.Errorf("git not installed: %v", err.Error())
		}
	}
	p := os.Getenv("PATH")
	pv := strings.Split(p, ";")
	pvv := make([]string, 0, len(pv)+2)
	if len(gitbin) != 0 {
		pvv = append(pvv, filepath.Clean(gitbin))
	}
	for _, s := range pv {
		if len(s) == 0 {
			continue
		}
		pvv = append(pvv, filepath.Clean(s))
	}
	os.Setenv("PATH", strings.Join(pvv, ";"))
	return nil
}

func init() {
	if err := initializeEnv(); err != nil {
		log.Logf("unable initialize env: %v", err)
	}
}
