// +build !linux,!windows

package sdk

import "golang.org/x/sys/unix"

var SIGINFO = unix.SIGINFO
