package tat

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArrayContains(t *testing.T) {
	array1 := []string{"a", "b"}
	assert.True(t, ArrayContains(array1, "a"), "should be true, a should be find in array{a, b}")
}

func TestArrayNotContains(t *testing.T) {
	array1 := []string{"a", "b", "c"}
	assert.False(t, ArrayContains(array1, "d"), "should be false")
}

func TestItemInBothArrays(t *testing.T) {
	array1 := []string{"a", "b", "c"}
	array2 := []string{"c", "d", "d"}
	assert.True(t, ItemInBothArrays(array1, array2), "should be true")
}

func TestItemInBothArraysFalse(t *testing.T) {
	array1 := []string{"a", "b", "c"}
	array2 := []string{"d", "e", "f"}
	assert.False(t, ItemInBothArrays(array1, array2), "should be false")
}
