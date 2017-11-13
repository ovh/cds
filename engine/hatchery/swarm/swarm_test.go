package swarm

import (
	"reflect"
	"testing"

	"github.com/fsouza/go-dockerclient"

	"github.com/ovh/cds/sdk"
)

func Test_computeDockerOpts(t *testing.T) {
	type args struct {
		isSharedInfra bool
		requirements  []sdk.Requirement
	}
	tests := []struct {
		name    string
		args    args
		want    *dockerOpts
		wantErr bool
	}{
		{
			name:    "Empty",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1"}}},
			want:    &dockerOpts{},
			wantErr: false,
		},
		{
			name:    "Simple Test with Ports",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081/tcp"}}},
			want:    &dockerOpts{ports: map[docker.Port][]docker.PortBinding{docker.Port("8081/tcp"): []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}}},
			wantErr: false,
		},
		{
			name:    "Ports with error",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8081/tcp"}}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Shared Infra",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8081/tcp"}}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Simple Test with Priviledge",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --priviledge"}}},
			want:    &dockerOpts{priviledge: true},
			wantErr: false,
		},
		{
			name: "Simple Test with Priviledge and two ports",
			args: args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081/tcp --priviledge --port=9080:9081/tcp"}}},
			want: &dockerOpts{priviledge: true, ports: map[docker.Port][]docker.PortBinding{
				docker.Port("8081/tcp"): []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
				docker.Port("9081/tcp"): []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: "9080"}},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := computeDockerOpts(tt.args.isSharedInfra, tt.args.requirements)
			if (err != nil) != tt.wantErr {
				t.Errorf("computeDockerOpts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeDockerOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}
