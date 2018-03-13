package swarm

import (
	"reflect"
	"testing"

	"github.com/docker/go-connections/nat"

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
			want:    &dockerOpts{ports: nat.PortMap{nat.Port("8081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}}},
			wantErr: false,
		},
		{
			name:    "Simple Test with Ports, without tcp, tcp is the default",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081"}}},
			want:    &dockerOpts{ports: nat.PortMap{nat.Port("8081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}}},
			wantErr: false,
		},
		{
			name:    "Simple Test with Ports, with udp",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081/udp"}}},
			want:    &dockerOpts{ports: nat.PortMap{nat.Port("8081/udp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}}},
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
			name:    "Simple Test with privileged",
			args:    args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --privileged"}}},
			want:    &dockerOpts{privileged: true},
			wantErr: false,
		},
		{
			name: "Simple Test with privileged and two ports",
			args: args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081/tcp --privileged --port=9080:9081/tcp"}}},
			want: &dockerOpts{privileged: true, ports: nat.PortMap{
				nat.Port("8081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
				nat.Port("9081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9080"}},
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
