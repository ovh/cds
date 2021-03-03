package hatchery

import (
	"strings"
	"testing"
)

func Test_generateWorkerName(t *testing.T) {
	type args struct {
		hatcheryName string
		isRegister   bool
		model        string
	}
	tests := []struct {
		name       string
		args       args
		wantPrefix string
	}{
		{
			name:       "simple",
			args:       args{hatcheryName: "p999-prod", isRegister: true, model: "rust-official-1.41"},
			wantPrefix: "register-rust-official-1-41-",
		},
		{
			name:       "simple special char",
			args:       args{hatcheryName: "p999/prod", isRegister: true, model: "shared.infra-rust-official-1.41"},
			wantPrefix: "register-shared-infra-rust-official-1-41-",
		},
		{
			name:       "long hatchery name",
			args:       args{hatcheryName: "p999-prod-xxxx-xxxx-xxxx-xxxx-xxxx", isRegister: true, model: "shared.infra-rust-official-1.41"},
			wantPrefix: "register-shared-infra-rust-official-1-41-",
		},
		{
			name:       "long model name",
			args:       args{hatcheryName: "hname", isRegister: true, model: "shared.infra-rust-official-1.41-xxx-xxx-xxx-xxx"},
			wantPrefix: "register-shared-infra-rust-official-1-41-xxx-xxx-xxx-x-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateWorkerName(tt.args.hatcheryName, tt.args.isRegister, tt.args.model)
			if len(got) > 64 {
				t.Errorf("len must be < 64() = %d - got:%s", len(got), got)
			}

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("generateWorkerName() = %v, want prefix : %v", got, tt.wantPrefix)
			}
		})
	}
}
