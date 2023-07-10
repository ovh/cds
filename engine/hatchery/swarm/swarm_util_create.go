package swarm

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/telemetry"
)

// create the docker bridge
func (h *HatcherySwarm) createNetwork(ctx context.Context, dockerClient *dockerClient, name string) error {
	ctx, end := telemetry.Span(ctx, "swarm.createNetwork", telemetry.Tag("network", name))
	defer end()
	log.Debug(ctx, "hatchery> swarm> createNetwork> Create network %s", name)
	_, err := dockerClient.NetworkCreate(ctx, name, types.NetworkCreate{
		Driver:         "bridge",
		Internal:       false,
		CheckDuplicate: true,
		EnableIPv6:     h.Config.NetworkEnableIPv6,
		IPAM: &network.IPAM{
			Driver: "default",
		},
		Labels: map[string]string{
			"worker_net": name,
		},
	})
	return err
}

type containerArgs struct {
	name, image, network, networkAlias string
	cmd, env                           []string
	labels                             map[string]string
	memory                             int64
	memorySwap                         int64
	dockerOpts                         dockerOpts
	entryPoint                         strslice.StrSlice
}

// shortcut to create+start(=run) a container
func (h *HatcherySwarm) createAndStartContainer(ctx context.Context, dockerClient *dockerClient, cArgs containerArgs, spawnArgs hatchery.SpawnArguments) error {
	if spawnArgs.Model.ModelV1 == nil && spawnArgs.Model.ModelV2 == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	ctx, end := telemetry.Span(ctx, "swarm.createAndStartContainer", telemetry.Tag(telemetry.TagWorker, cArgs.name))
	defer end()

	//Memory is set to 1GB by default
	if cArgs.memory <= 4 {
		cArgs.memory = 1024
	}
	log.Info(ctx, "create container %s on %s from %s (memory=%dMB)", cArgs.name, dockerClient.name, cArgs.image, cArgs.memory)

	var exposedPorts nat.PortSet

	name := cArgs.name
	config := &container.Config{
		Image:        cArgs.image,
		Env:          cArgs.env,
		Cmd:          cArgs.cmd,
		Labels:       cArgs.labels,
		ExposedPorts: exposedPorts,
	}

	if cArgs.entryPoint != nil {
		config.Entrypoint = cArgs.entryPoint
	}

	hostConfig := &container.HostConfig{
		PortBindings: cArgs.dockerOpts.ports,
		Privileged:   cArgs.dockerOpts.privileged,
		Mounts:       cArgs.dockerOpts.mounts,
		ExtraHosts:   cArgs.dockerOpts.extraHosts,
	}
	hostConfig.Resources = container.Resources{
		Memory:     cArgs.memory * 1024 * 1024, //from MB to B
		MemorySwap: cArgs.memorySwap,
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}

	if cArgs.network != "" && len(cArgs.networkAlias) > 0 {
		networkingConfig.EndpointsConfig[cArgs.network] = &network.EndpointSettings{
			Aliases: []string{cArgs.networkAlias, cArgs.name},
		}
	}

	hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryStartDockerPull, h.Name(), cArgs.image))

	_, next := telemetry.Span(ctx, "swarm.dockerClient.pullImage", telemetry.Tag("image", cArgs.image))
	if err := h.pullImage(dockerClient,
		cArgs.image,
		timeoutPullImage,
		spawnArgs.Model); err != nil {
		next()

		spawnMsg := sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryEndDockerPullErr, h.Name(), cArgs.image, sdk.Cause(err))
		hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, spawnMsg)
		return sdk.WrapError(err, "unable to pull image %s on %s", cArgs.image, dockerClient.name)
	}
	next()

	hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryEndDockerPull, h.Name(), cArgs.image))

	_, next = telemetry.Span(ctx, "swarm.dockerClient.ContainerCreate", telemetry.Tag(telemetry.TagWorker, cArgs.name), telemetry.Tag("network", fmt.Sprintf("%v", networkingConfig)))
	c, err := dockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		next()
		return sdk.WrapError(err, "unable to create container %s on %s", name, dockerClient.name)
	}
	next()

	_, next = telemetry.Span(ctx, "swarm.dockerClient.ContainerStart", telemetry.Tag(telemetry.TagWorker, cArgs.name), telemetry.Tag("network", fmt.Sprintf("%v", networkingConfig)))
	if err := dockerClient.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		next()
		return sdk.WrapError(err, "unable to start container on %s: %s", dockerClient.name, sdk.StringFirstN(c.ID, 12))
	}
	next()
	return nil
}

var regexPort = regexp.MustCompile("^--port=(.*):(.*)$")

type dockerOpts struct {
	ports      nat.PortMap
	privileged bool
	mounts     []mount.Mount
	extraHosts []string
}

func (h *HatcherySwarm) computeDockerOpts(requirements []sdk.Requirement) (*dockerOpts, error) {
	dockerOpts := &dockerOpts{}

	// support for add-host on hatchery configuration
	for _, opt := range strings.Split(h.Config.DockerOpts, " ") {
		if strings.HasPrefix(opt, "--add-host=") {
			if err := dockerOpts.computeDockerOptsExtraHosts(opt); err != nil {
				return nil, err
			}
		} else if opt == "--privileged" {
			dockerOpts.privileged = true
		}
	}

	for _, r := range requirements {
		switch r.Type {
		case sdk.ModelRequirement:
			if err := h.computeDockerOptsOnModelRequirement(dockerOpts, r); err != nil {
				return nil, err
			}
		}
	}

	return dockerOpts, nil
}

func (h *HatcherySwarm) computeDockerOptsOnModelRequirement(d *dockerOpts, req sdk.Requirement) error {
	// args are separated by a space
	// example: myGroup/golang:1.9.1 --port=8080:8080/tcp
	for idx, opt := range strings.Split(req.Value, " ") {
		if idx == 0 || strings.TrimSpace(opt) == "" {
			continue // it's image name
		}

		if h.Config.DisableDockerOptsOnRequirements {
			return fmt.Errorf("you could not use this docker options '%s' with a 'shared.infra' hatchery. Please use you own hatchery or remove this option", opt)
		}

		if strings.HasPrefix(opt, "--port=") {
			if err := d.computeDockerOptsPorts(opt); err != nil {
				return err
			}
		} else if strings.HasPrefix(opt, "--add-host=") {
			if err := d.computeDockerOptsExtraHosts(opt); err != nil {
				return err
			}
		} else if opt == "--privileged" {
			d.privileged = true
		} else {
			return fmt.Errorf("options not supported: %s", opt)
		}
	}
	return nil
}

func (d *dockerOpts) computeDockerOptsExtraHosts(arg string) error {
	value := strings.TrimPrefix(strings.TrimSpace(arg), "--add-host=")
	d.extraHosts = append(d.extraHosts, value)
	return nil
}

func (d *dockerOpts) computeDockerOptsPorts(arg string) error {
	if regexPort.MatchString(arg) {
		s := regexPort.FindStringSubmatch(arg)
		//s = --port=8081:8182/tcp // hostPort:containerPort
		//s[0] = --port=8081:8182/tcp
		//s[1] = 8081 // hostPort
		//s[2] = 8182/tcp  // containerPort
		containerPort := s[2]
		if !strings.Contains(containerPort, "/") {
			// tcp is the default
			containerPort += "/tcp"
		}
		if d.ports == nil {
			d.ports = nat.PortMap{}
		}
		if _, ok := d.ports[nat.Port(containerPort)]; !ok {
			d.ports[nat.Port(containerPort)] = []nat.PortBinding{}
		}
		//  "8182/tcp": {{HostIP: "0.0.0.0", HostPort: "8081"}}
		d.ports[nat.Port(containerPort)] = append(d.ports[nat.Port(containerPort)],
			nat.PortBinding{HostIP: "0.0.0.0", HostPort: s[1]})
		return nil // no error
	}
	return fmt.Errorf("wrong format of ports arguments. Example: --port=8081:8182/tcp")
}
