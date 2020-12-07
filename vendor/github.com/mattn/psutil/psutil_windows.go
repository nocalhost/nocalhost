package psutil

import (
	"syscall"
	"unsafe"
)

const PROCESS_ALL_ACCESS = 0x1F0FFF

var (
	kernel32         = syscall.NewLazyDLL("kernel32")
	procGetProcessId = kernel32.NewProc("GetProcessId")
)

func killTree(ph syscall.Handle, code uint32) error {
	var pe syscall.ProcessEntry32
	pid, _, _ := procGetProcessId.Call(uintptr(ph))
	if pid != 0 {
		h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
		if err == nil {
			pe.Size = uint32(unsafe.Sizeof(pe))
			if syscall.Process32First(h, &pe) == nil {
				for {
					if pe.ParentProcessID == uint32(pid) {
						ph, err := syscall.OpenProcess(
							PROCESS_ALL_ACCESS, false, pe.ProcessID)
						if err == nil {
							killTree(ph, code)
							syscall.CloseHandle(ph)
						}
					}
					if syscall.Process32Next(h, &pe) != nil {
						break
					}
				}
				syscall.CloseHandle(h)
			}
		}
	}
	return syscall.TerminateProcess(syscall.Handle(ph), uint32(code))
}

func Terminate(pid int, code int) error {
	ph, err := syscall.OpenProcess(
		PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(ph)
	return syscall.TerminateProcess(syscall.Handle(ph), uint32(code))
}

func TerminateTree(pid int, code int) error {
	ph, err := syscall.OpenProcess(
		PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(ph)
	return killTree(ph, uint32(code))
}
