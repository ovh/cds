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

func TestLuaCheckStringsFind(t *testing.T) {
	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"git_branch": "release/foo",
	})
	test.NoError(t, l.Perform(`return git_branch:find("^release/") ~= nil`))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckWeekOfDay(t *testing.T) {
	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	test.NoError(t, l.Perform(`return os.date("%w") < "8"`))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaNilVariables(t *testing.T) {
	script := `local re = require("re")
	return 
		(git_branch ~= nil and re.match(git_branch,"integration") == "integration") or 
		(git_repository ~= nil and re.match(git_repository,"integration") == "integration") or 
		(git_pr_title ~= nil and re.match(git_pr_title,"integration") == "integration")`

	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"git_repository": "PROJECT/integration",
	})
	test.NoError(t, l.Perform(script))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckRegularExpression(t *testing.T) {
	l, err := NewCheck()
	test.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	test.NoError(t, l.Perform(`
		local re = require("re")

		return re.match("abcdefg", "abc.*") == "abcdefg"
	`))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)

	test.NoError(t, l.Perform(`
		local re = require("re")

		return re.match("abcdefg", "zzz.*") == ""
	`))
	assert.False(t, l.IsError)
	assert.False(t, l.Result)

}
