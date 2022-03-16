/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package fp

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	pathSeparator = string(os.PathSeparator)

	parentDir  = ".."
	currentDir = "."
	empty      = ""
)

type FilePathEnhance struct {

	// originPath
	Path string

	// absPath
	absPath string

	cached  bool
	content string
	mu      sync.Mutex
}

func NewRandomTempPath() *FilePathEnhance {
	dir, _ := ioutil.TempDir("", "")
	return NewFilePath(dir)
}

func NewFilePath(path string) *FilePathEnhance {
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = home + path[1:]
		}
	}

	absPath, _ := filepath.Abs(path)

	return &FilePathEnhance{
		Path:    path,
		absPath: absPath,
	}
}

func (f *FilePathEnhance) Abs() string {
	return f.absPath
}

func (f *FilePathEnhance) RelOrAbs(path string) *FilePathEnhance {
	if filepath.IsAbs(path) {
		return NewFilePath(path)
	}

	// rel path
	baseSplited := strings.Split(f.absPath, pathSeparator)

	splited := strings.Split(path, pathSeparator)
	for _, s := range splited {
		switch s {
		case empty:
		case currentDir:
			break
		case parentDir:
			if len(baseSplited) == 0 {
				break
			}

			baseSplited = baseSplited[:len(baseSplited)-1]
			break
		default:
			baseSplited = append(baseSplited, s)
		}
	}

	return NewFilePath(strings.Join(baseSplited, pathSeparator))
}

func (f *FilePathEnhance) WriteFile(value string) error {
	f.mu.Lock()

	defer func() {
		f.cached = false
		f.mu.Unlock()
	}()

	return ioutil.WriteFile(f.absPath, []byte(value), os.FileMode(0600))
}

func (f *FilePathEnhance) ReadFile() string {
	f.mu.Lock()

	defer func() {
		f.cached = true
		f.mu.Unlock()
	}()

	if !f.cached {

		b, err := ioutil.ReadFile(f.absPath)
		if err != nil {

		} else {
			f.content = string(b)
		}
	}

	return f.content
}

func (f *FilePathEnhance) Doom() error {
	return os.Remove(f.absPath)
}

func (f *FilePathEnhance) ReadFileCompel() (string, error) {
	f.mu.Lock()

	defer func() {
		f.cached = true
		f.mu.Unlock()
	}()

	if !f.cached {

		b, err := ioutil.ReadFile(f.absPath)
		if err != nil {
			return "", err
		} else {
			f.content = string(b)
		}
	}

	return f.content, nil
}

func (f *FilePathEnhance) CheckExist() error {
	_, err := os.Stat(f.absPath)
	if err != nil {
		_, err := os.Stat(f.absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("File %s not found.", f.absPath))
			}
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (f *FilePathEnhance) Mkdir() error {
	return os.MkdirAll(f.absPath, 0700)
}

func (f *FilePathEnhance) MkdirThen() *FilePathEnhance {
	_ = os.MkdirAll(f.absPath, 0700)
	return f
}

func (f *FilePathEnhance) ReadEnvFile() []string {
	var envFiles []string

	if f == nil {
		return envFiles
	}

	if err := f.CheckExist(); err != nil {
		return envFiles
	}

	file, err := os.Open(f.Abs())
	if err != nil {
		log.ErrorE(err, "Can't not reading env file from "+f.Path)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if !strings.ContainsAny(text, "=") || strings.HasPrefix(text, "#") {
			continue
		}
		envFiles = append(envFiles, text)
	}
	return envFiles
}

func (f *FilePathEnhance) ReadEnvFileKV() map[string]string {
	envFiles := make(map[string]string, 0)

	if f == nil {
		return envFiles
	}

	if err := f.CheckExist(); err != nil {
		return envFiles
	}

	file, err := os.Open(f.Abs())
	if err != nil {
		log.ErrorE(err, "Can't not reading env file from "+f.Path)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if !strings.ContainsAny(text, "=") || strings.HasPrefix(text, "#") {
			continue
		}
		index := strings.Index(text, "=")

		if len(text)-1 == index {
			return nil
		}

		envFiles[text[:index]] = text[index+1:]
	}
	return envFiles
}

func (f *FilePathEnhance) Remove()error {
	return os.Remove(f.absPath)
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		errStr := fmt.Sprintf("Failed to getting from %s, err: %d", url, resp.StatusCode)
		bs, err := req.ResponseToBArray(resp)
		if err != nil {
			return nil, fmt.Errorf(errStr)
		}
		body := string(bs)
		if body != "" {
			errStr += "\nBody: " + body
		}
		return nil, fmt.Errorf(errStr)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return bs, nil
}
