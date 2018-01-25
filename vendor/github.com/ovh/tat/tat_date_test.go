package tat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitFloatForTimeUnix(t *testing.T) {
	a, b := SplitFloatForTimeUnix(12345678.887766)
	assert.True(t, a == 12345678, "a should be equals to 12345678")
	assert.True(t, b == 887766000, "b should be equals to 887766000")
}

func TestSplitFloatForTimeUnixSec(t *testing.T) {
	a, b := SplitFloatForTimeUnix(12345678)
	assert.True(t, a == 12345678, "a should be equals to 12345678")
	assert.True(t, b == 0, "b should be equals to 0")
}
