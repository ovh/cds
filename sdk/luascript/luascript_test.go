package luascript

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/assert"
)

func TestLuaCheck(t *testing.T) {
	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	test.NoError(t, l.Perform("return cds_application == \"mon-appli\""))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckStrings(t *testing.T) {
	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	test.NoError(t, l.Perform("return string.match(\"abcdefg\", \"b..\") == \"bcd\""))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}
