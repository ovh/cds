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
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{hatcheryName: "p999-prod", isRegister: true, model: "shared.infra-rust-official-1.41"},
			want: "register-p999-prod-shared-infra-ru",
		},
		{
			name: "simple special char",
			args: args{hatcheryName: "p999/prod", isRegister: true, model: "shared.infra-rust-official-1.41"},
			want: "register-p999-prod-shared-infra-ru",
		},
		{
			name: "long hatchery name",
			args: args{hatcheryName: "p999-prod-xxxx-xxxx-xxxx-xxxx-xxxx", isRegister: true, model: "shared.infra-rust-official-1.41"},
			want: "register-shared-infra-ru",
		},
		{
			name: "long model name",
			args: args{hatcheryName: "hname", isRegister: true, model: "shared.infra-rust-official-1.41-xxx-xxx-xxx-xxx"},
			want: "register-hname-shared-infra-ru",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateWorkerName(tt.args.hatcheryName, tt.args.isRegister, tt.args.model)
			if len(got) > 64 {
				t.Errorf("len must be < 64() = %d - got:%s", len(got), got)
			}

			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("generateWorkerName() = %v, want prefix : %v", got, tt.want)
			}
		})
	}
}
