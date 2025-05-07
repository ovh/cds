package sdk

import "testing"

func TestIsGPGKeyAlreadyInstalled(t *testing.T) {
	type args struct {
		longKeyID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should not exist",
			args: args{"B9A01AB2DFD257E8AC279E929C5B5EE03BD470B7"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGPGKeyAlreadyInstalled(tt.args.longKeyID); got != tt.want {
				t.Errorf("IsGPGKeyAlreadyInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}
