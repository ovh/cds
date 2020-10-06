package sdk

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringIsAscii(t *testing.T) {
	assert.True(t, StringIsAscii("aaa"))
	assert.False(t, StringIsAscii("aaa 🚀"))
}

func TestRemoveNotPrintableChar(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{
			in:  "test",
			out: "test",
		},
		{
			in:  "test" + string([]byte{0x00}),
			out: "test ",
		},
		{
			in:  "test" + string([]byte{0xbf}),
			out: "test ",
		},
	}

	for i := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tests[i].out, RemoveNotPrintableChar(tests[i].in))
		})
	}
}

func TestPathIsAbs(t *testing.T) {
	GOOS = "windows"
	assert.True(t, PathIsAbs(`C:\Program Files (x86)\Foo`))
	assert.False(t, PathIsAbs(`Program Files (x86)\Foo`))
	GOOS = "linux"
	assert.True(t, PathIsAbs(`/tmp`))
	assert.False(t, PathIsAbs(`tmp`))
}
