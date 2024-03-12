package namesgenerator

import (
	"strings"
	"testing"
)

func TestNameFormat(t *testing.T) {
	name := GetRandomNameCDS()
	if !strings.Contains(name, "_") {
		t.Fatalf("Generated name does not contain an underscore")
	}
	t.Log("name generated:", name)
	if strings.ContainsAny(name, "0123456789") {
		t.Fatalf("Generated name contains numbers!")
	}
}

func TestGetRandomNameCDSWithMaxLength(t *testing.T) {
	type args struct {
		maxLength int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "nominal_case",
			args: args{maxLength: 20},
		},
		{
			name: "with_short_length",
			args: args{maxLength: 9},
		},
		{
			name: "with_very_short_length",
			args: args{maxLength: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRandomNameCDSWithMaxLength(tt.args.maxLength)
			t.Logf("got: %v", got)
			if len(got) > tt.args.maxLength {
				t.Errorf("GetRandomNameCDSWithMaxLength() = %v", got)
			}
		})
	}
}
