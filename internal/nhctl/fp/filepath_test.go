package fp

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	pathSeparator = string(os.PathSeparator)

	parentDir  = ".."
	currentDir = "."
	empty      = ""
)

type FilePath struct {
	absPath string
}

func NewFilePath(path string) (*FilePath, error) {
	abs, err := filepath.Abs(path)

	if err != nil {
		return nil, err
	}

	return &FilePath{
		absPath: abs,
	}, nil
}

func (f *FilePath) relOrAbs(path string) (*FilePath, error) {
	if path[0] == os.PathSeparator {
		return NewFilePath(path)
	}

	baseSplited := strings.Split(f.absPath, pathSeparator)

	splited := strings.Split(path, pathSeparator)
	for _, s := range splited {
		switch s {
		case empty:
		case currentDir:
			break
		case parentDir:
			if len(baseSplited) == 0 {
				// 报错
			}

			baseSplited = baseSplited[:len(baseSplited)-1]
			break
		default:
			baseSplited = append(baseSplited, s)
		}
	}

	return NewFilePath(strings.Join(baseSplited, pathSeparator))
}
