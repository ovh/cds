package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseHeaders(t *testing.T) {
	headers := parseHeaders(`"X-OVH-KEY"="test"
"Authorization"="Basic blablabla==&d"
"EXAMPLE"="HASH"`)

	assert.Equal(t, headers.Get("X-OVH-KEY"), "test")
	assert.Equal(t, headers.Get("Authorization"), "Basic blablabla==&d")
	assert.Equal(t, headers.Get("EXAMPLE"), "HASH")
}
