package main

import (
	"log"
	"strings"

	"github.com/ovh/cds/sdk/plugin"
)

//Logf is a wrapper to plugin.sendLog
func Logf(format string, args ...interface{}) {
	if strings.TrimSpace(format) == "" {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	plugin.SendLog(job, format, args...)
	log.Printf(format, args...)
}
