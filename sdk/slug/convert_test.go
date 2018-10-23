package slug

import (
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name, value, result string
	}{
		{
			name:   "Text already a slug",
			value:  "hello-world",
			result: "hello-world",
		},
		{
			name:   "With spaces and special caracters",
			value:  "Hello World !",
			result: "hello-world",
		},
		{
			name:   "With spaces around",
			value:  "    Hello World !    ",
			result: "hello-world",
		},
		{
			name:   "Only special caracters",
			value:  "    &+=:/.;?,\"'(§!)    ",
			result: "",
		},
		{
			name:   "Convert accent",
			value:  "éàçÎEEÉèⓩ",
			result: "eacieeeez",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			slug := Convert(test.value)
			if slug != test.result {
				t.Errorf("Convert(%s) = %v, want %v", test.value, slug, test.result)
			}
		})
	}
}
