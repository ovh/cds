package swarm

import (
	"reflect"
	"testing"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
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
		{
			name: "Simple Test with volume",
			args: args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.VolumeRequirement, Value: "type=bind,source=/hostDir/sourceDir,destination=/dirInJob"}}},
			want: &dockerOpts{
				mounts: []mount.Mount{
					{
						Type:   mount.TypeBind,
						Source: "/hostDir/sourceDir",
						Target: "/dirInJob",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Simple Test with readonly volume",
			args: args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.VolumeRequirement, Value: "type=bind,source=/hostDir/sourceDir,destination=/dirInJob,readonly"}}},
			want: &dockerOpts{
				mounts: []mount.Mount{
					{
						Type:     mount.TypeBind,
						Source:   "/hostDir/sourceDir",
						Target:   "/dirInJob",
						ReadOnly: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Extra hosts",
			args: args{requirements: []sdk.Requirement{{Name: "go-official-1.9.1", Type: sdk.ModelRequirement, Value: "golang:1.9.1 --port=8080:8081/tcp --privileged --port=9080:9081/tcp --add-host=aaa:1.2.3.4 --add-host=bbb:5.6.7.8"}}},
			want: &dockerOpts{
				privileged: true,
				ports: nat.PortMap{
					nat.Port("8081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
					nat.Port("9081/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9080"}},
				},
				extraHosts: []string{"aaa:1.2.3.4", "bbb:5.6.7.8"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HatcherySwarm{hatch: &sdk.Hatchery{IsSharedInfra: tt.args.isSharedInfra}}
			got, err := h.computeDockerOpts(tt.args.requirements)
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

func TestHatcherySwarm_createAndStartContainer(t *testing.T) {
	h := testSwarmHatchery(t)
	args := containerArgs{
		name:   "my-nginx",
		image:  "nginx:latest",
		env:    []string{"FROM_CDS", "FROM_CDS"},
		labels: map[string]string{"FROM_CDS": "FROM_CDS"},
		memory: 256,
	}

	// RegisterOnly = true, this will pull image if image is not found
	spawnArgs := hatchery.SpawnArguments{RegisterOnly: true}
	err := h.createAndStartContainer(h.dockerClients["default"], args, spawnArgs)
	test.NoError(t, err)

	cntr, err := h.getContainer(h.dockerClients["default"], args.name, types.ContainerListOptions{})
	test.NoError(t, err)

	err = h.killAndRemove(h.dockerClients["default"], cntr.ID)
	test.NoError(t, err)
}

func TestHatcherySwarm_createAndStartContainerWithMount(t *testing.T) {
	h := testSwarmHatchery(t)
	args := containerArgs{
		name:   "my-nginx",
		image:  "nginx:latest",
		cmd:    []string{"uname"},
		env:    []string{"FROM_CDS", "FROM_CDS"},
		labels: map[string]string{"FROM_CDS": "FROM_CDS"},
		memory: 256,
		dockerOpts: dockerOpts{
			mounts: []mount.Mount{
				{
					Source:   "/tmp",
					Target:   "/tmp",
					Type:     mount.TypeBind,
					ReadOnly: true,
					BindOptions: &mount.BindOptions{
						Propagation: mount.PropagationRPrivate,
					},
				},
			},
		},
	}

	err := h.pullImage(h.dockerClients["default"], args.image, timeoutPullImage)
	test.NoError(t, err)

	spawnArgs := hatchery.SpawnArguments{RegisterOnly: false}
	err = h.createAndStartContainer(h.dockerClients["default"], args, spawnArgs)
	test.NoError(t, err)

	cntr, err := h.getContainer(h.dockerClients["default"], args.name, types.ContainerListOptions{})
	test.NoError(t, err)

	err = h.killAndRemove(h.dockerClients["default"], cntr.ID)
	test.NoError(t, err)
}

func TestHatcherySwarm_createAndStartContainerWithNetwork(t *testing.T) {
	h := testSwarmHatchery(t)
	args := containerArgs{
		name:         "my-nginx",
		image:        "nginx:latest",
		cmd:          []string{"uname"},
		env:          []string{"FROM_CDS", "FROM_CDS"},
		labels:       map[string]string{"FROM_CDS": "FROM_CDS"},
		memory:       256,
		network:      "my-network",
		networkAlias: "my-container",
	}

	err := h.createNetwork(h.dockerClients["default"], args.network)
	test.NoError(t, err)

	spawnArgs := hatchery.SpawnArguments{RegisterOnly: false}
	err = h.createAndStartContainer(h.dockerClients["default"], args, spawnArgs)
	test.NoError(t, err)

	cntr, err := h.getContainer(h.dockerClients["default"], args.name, types.ContainerListOptions{})
	test.NoError(t, err)

	err = h.killAndRemove(h.dockerClients["default"], cntr.ID)
	test.NoError(t, err)
}
