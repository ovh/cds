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

func Test_GenerateWorkerName(t *testing.T) {
	type args struct {
		prefix string
		model  string
	}
	tests := []struct {
		name       string
		args       args
		wantPrefix string
	}{
		{
			name:       "simple",
			args:       args{prefix: "register", model: "rust-official-1.41"},
			wantPrefix: "register-rust-official-1-41",
		},
		{
			name:       "simple special char",
			args:       args{prefix: "register", model: "shared.infra-rust-official-1.41"},
			wantPrefix: "register-shared-infra-rust-official-1-41",
		},
		{
			name:       "long hatchery name",
			args:       args{prefix: "register", model: "shared.infra-rust-official-1.41"},
			wantPrefix: "register-shared-infra-rust-official-1-41",
		},
		{
			name:       "long model name",
			args:       args{prefix: "register", model: "shared.infra-rust-official-1.41-xxx-xxx-xxx-xxx"},
			wantPrefix: "register-shared-infra-rust-official-1-41-xxx-xxx-xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWorkerName(tt.args.model, tt.args.prefix)
			if len(got) > 63 {
				t.Errorf("len must be < 63() = %d - got: %s", len(got), got)
			}

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("GenerateWorkerName() = %s, want prefix: %s", got, tt.wantPrefix)
			}
		})
	}
}
