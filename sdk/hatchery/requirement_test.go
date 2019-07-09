package hatchery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk/hatchery"
)

func TestParseRequirementModel(t *testing.T) {
	for _, test := range []struct {
		in          string
		expectedImg string
		expectedEnv map[string]string
	}{
		{
			in:          "",
			expectedImg: "",
			expectedEnv: nil,
		},
		{
			in:          "no:env",
			expectedImg: "no:env",
			expectedEnv: nil,
		},
		{
			in:          "no:env:space   ",
			expectedImg: "no:env:space",
			expectedEnv: nil,
		},
		{
			in:          `image_name TEST=abc V1='a "b" c' V2="a 'b' c" V3 V4=`,
			expectedImg: "image_name",
			expectedEnv: map[string]string{
				"TEST": "abc",
				`V1`:   `a "b" c`,
				"V2":   "a 'b' c",
				"V3":   "",
				"V4":   "",
			},
		},
		{
			in:          `with_spaces   TEST="foo  bar"   V1=12   V2= V3 V4='1  8'`,
			expectedImg: "with_spaces",
			expectedEnv: map[string]string{
				"TEST": "foo  bar",
				"V1":   "12",
				"V2":   "",
				"V3":   "",
				"V4":   "1  8",
			},
		},
		{
			in:          `backslash FOO="a\"\b\\c"  BAR='a\'\b\\c'`,
			expectedImg: "backslash",
			expectedEnv: map[string]string{
				`FOO`: `a"b\c`,
				`BAR`: `a'b\c`,
			},
		},
	} {
		img, env := hatchery.ParseRequirementModel(test.in)

		assert.Equal(t, test.expectedImg, img,
			"image returned by ParseRequirementModel("+test.in+")")

		assert.Equal(t, test.expectedEnv, env,
			"environment returned by ParseRequirementModel("+test.in+")")
	}
}

func TestParseArgs(t *testing.T) {
	for _, test := range []struct {
		in           string
		expectedArgs []string
	}{
		{
			in:           "",
			expectedArgs: nil,
		},
		{
			in:           "  \t ",
			expectedArgs: nil,
		},
		{
			in:           "  abc \t\n   def ghi \t ",
			expectedArgs: []string{"abc", "def", "ghi"},
		},
		{
			in:           `  abc 'def' "ghi" `,
			expectedArgs: []string{"abc", "def", "ghi"},
		},
		{
			in:           `  abc d'e'f g"h"i `,
			expectedArgs: []string{"abc", "def", "ghi"},
		},
		{
			in:           `  abc '' "" `,
			expectedArgs: []string{"abc", "", ""},
		},
		{
			in:           `  a\\b\c\  'd\\e"\f\'' "g\\h'\i\"" `,
			expectedArgs: []string{`a\bc `, `d\e"f'`, `g\h'i"`},
		},
		{
			// edge case, non closed ", skip it anyway
			in:           ` "abc  `,
			expectedArgs: []string{`abc`},
		},
		{
			// edge case, non closed ', skip it anyway
			in:           ` 'abc  `,
			expectedArgs: []string{`abc`},
		},
		{
			// edge case, final backslash, keep it as is
			in:           ` 'abc\`,
			expectedArgs: []string{`abc\`},
		},
		{
			// typical use case
			in:           `--name="Bob Foo" -c user='foo bar' '' ""`,
			expectedArgs: []string{`--name=Bob Foo`, `-c`, `user=foo bar`, ``, ``},
		},
	} {
		args := hatchery.ParseArgs(test.in)

		assert.Equal(t, test.expectedArgs, args, "ParseArgs("+test.in+")")
	}
}
