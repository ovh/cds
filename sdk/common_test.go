package sdk

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
