//go:build darwin && arm64
// +build darwin,arm64

package bin

import (
	"embed"
	"io/ioutil"
)

//go:embed syncthing_macos_arm64
var f embed.FS

var binName = "syncthing_macos_arm64"

func CopyToBinPath(dst string) error {
	file, err := f.ReadFile(binName)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(dst, file, 0700); err != nil {
		return err
	}
	return nil
}
