package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

	hostConfig := &container.HostConfig{}
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

	if sdk.IsValidUUID(spawnArgs.JobID) {
		if err := h.CDSClientV2().V2QueuePushJobInfo(ctx, spawnArgs.Region, spawnArgs.JobID, sdk.V2SendJobRunInfo{
			Time:    time.Now(),
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("starting docker pull %s...", cArgs.image),
		}); err != nil {
			log.Warn(ctx, "unable to send job info for job %s: %v", spawnArgs.JobID, err)
		}
	} else {
		hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryStartDockerPull, h.Name(), cArgs.image))
	}

	_, next := telemetry.Span(ctx, "swarm.dockerClient.pullImage", telemetry.Tag("image", cArgs.image))
	if err := h.pullImage(dockerClient,
		cArgs.image,
		timeoutPullImage,
		spawnArgs.Model); err != nil {
		next()

		if sdk.IsValidUUID(spawnArgs.JobID) {
			if err := h.CDSClientV2().V2QueuePushJobInfo(ctx, spawnArgs.Region, spawnArgs.JobID, sdk.V2SendJobRunInfo{
				Time:    time.Now(),
				Level:   sdk.WorkflowRunInfoLevelError,
				Message: fmt.Sprintf("docker pull %s done with error: %v", cArgs.image, sdk.Cause(err)),
			}); err != nil {
				log.Warn(ctx, "unable to send job info for job %s: %v", spawnArgs.JobID, err)
			}
		} else {
			spawnMsg := sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryEndDockerPullErr, h.Name(), cArgs.image, sdk.Cause(err))
			hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, spawnMsg)
		}
		return sdk.WrapError(err, "unable to pull image %s on %s", cArgs.image, dockerClient.name)
	}
	next()

	if sdk.IsValidUUID(spawnArgs.JobID) {
		if err := h.CDSClientV2().V2QueuePushJobInfo(ctx, spawnArgs.Region, spawnArgs.JobID, sdk.V2SendJobRunInfo{
			Time:    time.Now(),
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Message: fmt.Sprintf("docker pull %s done", cArgs.image),
		}); err != nil {
			log.Warn(ctx, "unable to send job info for job %s: %v", spawnArgs.JobID, err)
		}
	} else {
		hatchery.SendSpawnInfo(ctx, h, spawnArgs.JobID, sdk.SpawnMsgNew(*sdk.MsgSpawnInfoHatcheryEndDockerPull, h.Name(), cArgs.image))
	}

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
