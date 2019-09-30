// +build windows

package sdk

import "syscall"

var SIGINFO = syscall.Signal(0x1d)
