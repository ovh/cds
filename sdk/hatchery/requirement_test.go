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
