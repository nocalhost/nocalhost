//go:build windows
// +build windows

package util

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"syscall"
)

//	ref https://stackoverflow.com/questions/31558066/how-to-ask-for-administer-privileges-on-windows-with-go
func RunWithElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := windows.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		logrus.Warn(err)
	}
}

func IsAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}
