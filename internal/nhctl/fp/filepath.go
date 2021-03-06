package fp

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"nocalhost/internal/nhctl/syncthing/network/req"
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

func NewFilePath(path string) *FilePathEnhance {
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
