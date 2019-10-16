package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_computeSemver(t *testing.T) {

	val1, err := computeSemver("1.6.2-1", "6")
	assert.NoError(t, err)
	assert.Equal(t, "1.6.2-1+cds.6", val1)

	val2, err2 := computeSemver("0.31.1-4-g595de235a", "6")
	assert.NoError(t, err2)
	assert.Equal(t, "0.31.1-4+sha.g595de235a.cds.6", val2)

	val3, err3 := computeSemver("0.31.1", "5")
	assert.NoError(t, err3)
	assert.Equal(t, "0.31.1+cds.5", val3)

}
