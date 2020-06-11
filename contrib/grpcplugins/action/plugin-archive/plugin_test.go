package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_Run(t *testing.T) {
	defer os.RemoveAll("self.tar.gz")
	defer os.RemoveAll("self/")

	subject := &archiveActionPlugin{}
	compressOpts := &actionplugin.ActionQuery{Options: map[string]string{
		"source":      ".",
		"destination": "self.tar.gz",
		"action":      "compress",
	}}

	result, err := subject.Run(context.Background(), compressOpts)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.FileExists(t, "self.tar.gz")

	uncompressOpts := &actionplugin.ActionQuery{Options: map[string]string{
		"source":      "self.tar.gz",
		"destination": "self/",
		"action":      "uncompress",
	}}
	result, err = subject.Run(context.Background(), uncompressOpts)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.FileExists(t, "self/main.go")
}
