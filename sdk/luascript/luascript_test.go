package luascript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuaCheck(t *testing.T) {
	l, err := NewCheck()
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	require.NoError(t, l.Perform("return cds_application == \"mon-appli\""))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaPerformErrorNoBoolReturn(t *testing.T) {
	l1, err := NewCheck()
	require.NoError(t, err)
	require.Error(t, l1.Perform(""))
	require.True(t, l1.IsError)
	require.False(t, l1.Result)

	l2, err := NewCheck()
	require.NoError(t, err)
	require.Error(t, l2.Perform("return nil"))
	require.True(t, l2.IsError)
	require.False(t, l2.Result)
}

func TestLuaCheckStrings(t *testing.T) {
	l, err := NewCheck()
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	require.NoError(t, l.Perform("return string.match(\"abcdefg\", \"b..\") == \"bcd\""))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckStringsFind(t *testing.T) {
	l, err := NewCheck()
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"git_branch": "release/foo",
	})
	require.NoError(t, l.Perform(`return git_branch:find("^release/") ~= nil`))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckWeekOfDay(t *testing.T) {
	l, err := NewCheck()
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	require.NoError(t, l.Perform(`return os.date("%w") < "8"`))
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
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"git_repository": "PROJECT/integration",
	})
	require.NoError(t, l.Perform(script))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)
}

func TestLuaCheckRegularExpression(t *testing.T) {
	l, err := NewCheck()
	require.NoError(t, err)
	l.SetVariables(map[string]string{
		"cds.application": "mon-appli",
	})
	require.NoError(t, l.Perform(`
		local re = require("re")

		return re.match("abcdefg", "abc.*") == "abcdefg"
	`))
	assert.False(t, l.IsError)
	assert.True(t, l.Result)

	require.NoError(t, l.Perform(`
		local re = require("re")

		return re.match("abcdefg", "zzz.*") == ""
	`))
	assert.False(t, l.IsError)
	assert.False(t, l.Result)
}

func Test_luaPerformStrictCheckOnVariable(t *testing.T) {
	luaCheck, err := NewCheck()
	require.NoError(t, err)

	luaCheck.SetVariables(map[string]string{
		"defined_string": "value",
	})
	luaCheck.SetFloatVariables(map[string]float64{
		"defined_number": 123,
	})

	require.NoError(t, luaCheck.EnableStrict())

	require.Error(t, luaCheck.Perform("return undefined_string == 'value'"))
	require.False(t, luaCheck.Result)
	require.Error(t, luaCheck.Perform("return undefined_string ~= 'value'"))
	require.False(t, luaCheck.Result)
	require.NoError(t, luaCheck.Perform("return defined_string == 'value'"))
	require.True(t, luaCheck.Result)
	require.NoError(t, luaCheck.Perform("return defined_string ~= 'value'"))
	require.False(t, luaCheck.Result)

	require.Error(t, luaCheck.Perform("return undefined_number < 1000"))
	require.False(t, luaCheck.Result)
	require.Error(t, luaCheck.Perform("return undefined_number < 100"))
	require.False(t, luaCheck.Result)
	require.NoError(t, luaCheck.Perform("return defined_number < 1000"))
	require.True(t, luaCheck.Result)
	require.NoError(t, luaCheck.Perform("return defined_number < 100"))
	require.False(t, luaCheck.Result)
}
