// +build amd64,linux

package bin

import (
	"embed"
	"io/fs"
	"io/ioutil"
)

//go:embed syncthing_linux_amd64
var f embed.FS

var BinName = "syncthing_linux_amd64"

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
