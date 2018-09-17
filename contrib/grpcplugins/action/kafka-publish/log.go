package main

import (
	"fmt"
	"strings"
)

//Logf is a wrapper to plugin.sendLog
func Logf(format string, args ...interface{}) {
	if strings.TrimSpace(format) == "" {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Printf(format, args...)
}
