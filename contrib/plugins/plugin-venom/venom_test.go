package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkGlobFile(t *testing.T) {
	files, err := walkGlobFile("tests/*")
	assert.Nil(t, err, "no error")
	assert.Equal(t, 3, len(files))

	files, err = walkGlobFile("tests/file1.yml")
	assert.Nil(t, err, "no error")
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "tests/file1.yml", files[0])
}
