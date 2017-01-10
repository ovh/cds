package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func NoError(t *testing.T, err error, msg ...interface{}) {
	assert.NoError(t, err)
	if err != nil {
		t.Fatal(msg...)
	}
}

func NotNil(t *testing.T, i interface{}, msg ...interface{}) {
	assert.NotNil(t, i)
	if i == nil {
		t.Fatal(msg...)
	}
}

func NotEmpty(t *testing.T, i interface{}, msg ...interface{}) {
	if !assert.NotEmpty(t, i) {
		t.Fatal(msg...)
	}
}
