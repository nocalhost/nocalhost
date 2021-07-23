// +build windows

package bin

import (
	"embed"
	"io/fs"
	"io/ioutil"
)

//go:embed syncthing_windows_amd64.exe
var f embed.FS

var BinName = "syncthing_windows_amd64.exe"

func CopyToBinPath(dst string) error {
	file, err := f.ReadFile(binName)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(dst, file, fs.ModePerm); err != nil {
		return err
	}
	return nil
}
