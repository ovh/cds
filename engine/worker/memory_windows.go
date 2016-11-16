// +build windows

package main

import (
	"syscall"
	"unsafe"
)

func systemTotalMemory() (uint64, error) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var proc = mod.NewProc("GetPhysicallyInstalledSystemMemory")
	var mem uint64

	_, _, err := proc.Call(uintptr(unsafe.Pointer(&mem)))
	if err != nil {
		return 0, nil
	}

	return mem, nil
}
