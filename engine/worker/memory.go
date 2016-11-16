// +build !windows

package main

// #include <unistd.h>
import (
	"C"
)

func systemTotalMemory() (uint64, error) {
	s := C.sysconf(C._SC_PHYS_PAGES) * C.sysconf(C._SC_PAGE_SIZE)
	return uint64(s), nil
}
