package utils

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
)

func GetHomePath() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		if u, err := user.Lookup(sudoUser); err == nil {
			return u.HomeDir
		}
	} else {
		u, err := user.Current()
		if err == nil {
			return u.HomeDir
		}
	}
	return ""
}

func IsSudoUser() bool {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return true
	}
	return false
}

func Sha1ToString(str string) string {
	hash := sha1.New()
	hash.Write([]byte(str))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func GetNhctlBinName() string {
	if runtime.GOOS == "windows" {
		return "nhctl.exe"
	}
	return "nhctl"
}

func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist
// Symlinks are ignored and skipped.
func CopyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	//if err == nil {
	//	return fmt.Errorf("destination already exists")
	//}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

func CheckKubectlVersion(compareMinor int) error {
	commonParams := []string{"version", "-o", "json"}
	jsonBody, err := tools.ExecCommand(nil, false, "kubectl", commonParams...)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonBody), &result)
	if err != nil {
		return err
	}
	targetResult := reflect.ValueOf(result["clientVersion"])
	target := targetResult.Interface().(map[string]interface{})
	minor, err := strconv.Atoi(target["minor"].(string))
	if err != nil {
		return err
	}
	if compareMinor > minor {
		return errors.New(fmt.Sprintf("kubectl version required %d+", compareMinor))
	}
	return nil
}
