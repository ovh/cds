package swarm

import (
	"fmt"
	"regexp"
	"strings"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

//create the docker bridge
func (h *HatcherySwarm) createNetwork(name string) error {
	log.Debug("createNetwork> Create network %s", name)
	_, err := h.dockerClient.NetworkCreate(context.Background(), name, types.NetworkCreate{
		Driver:         "bridge",
		Internal:       false,
		CheckDuplicate: true,
		EnableIPv6:     false,
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
	dockerOpts                         dockerOpts
	entryPoint                         strslice.StrSlice
}

//shortcut to create+start(=run) a container
func (h *HatcherySwarm) createAndStartContainer(cArgs containerArgs, spawnArgs hatchery.SpawnArguments) error {
	//Memory is set to 1GB by default
	if cArgs.memory <= 4 {
		cArgs.memory = 1024
	}
	log.Debug("createAndStartContainer> Create container %s from %s on network %s as %s (memory=%dMB)", cArgs.name, cArgs.image, cArgs.network, cArgs.networkAlias, cArgs.memory)

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
		MemorySwap: -1,
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}

	if cArgs.network != "" && len(cArgs.networkAlias) > 0 {
		networkingConfig.EndpointsConfig[cArgs.network] = &network.EndpointSettings{
			Aliases: []string{cArgs.networkAlias, cArgs.name},
		}
	}

	// ensure that image exists for register
	if spawnArgs.RegisterOnly {
		var images []types.ImageSummary
		var errl error
		images, errl = h.dockerClient.ImageList(context.Background(), types.ImageListOptions{All: true})
		if errl != nil {
			log.Warning("createAndStartContainer> Unable to list images: %s", errl)
		}

		var imageFound bool
	checkImage:
		for _, img := range images {
			for _, t := range img.RepoTags {
				if cArgs.image == t {
					imageFound = true
					break checkImage
				}
			}
		}

		if !imageFound {
			if err := h.pullImage(cArgs.image, timeoutPullImage); err != nil {
				return sdk.WrapError(err, "createAndStartContainer> Unable to pull image %s", cArgs.image)
			}
		}
	}

	c, err := h.dockerClient.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, name)
	if err != nil {
		return sdk.WrapError(err, "createAndStartContainer> Unable to create container %s", name)
	}

	if err := h.dockerClient.ContainerStart(context.Background(), c.ID, types.ContainerStartOptions{}); err != nil {
		return sdk.WrapError(err, "createAndStartContainer> Unable to start container %v %s", c.ID[:12])
	}
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
			if err := dockerOpts.computeDockerOptsOnModelRequirement(h.hatch.IsSharedInfra, r); err != nil {
				return nil, err
			}
		case sdk.VolumeRequirement:
			if err := dockerOpts.computeDockerOptsOnVolumeRequirement(h.hatch.IsSharedInfra, r); err != nil {
				return nil, err
			}
		}
	}

	return dockerOpts, nil
}

func (d *dockerOpts) computeDockerOptsOnModelRequirement(isSharedInfra bool, req sdk.Requirement) error {
	// args are separated by a space
	// example: golang:1.9.1 --port=8080:8080/tcp
	for idx, opt := range strings.Split(req.Value, " ") {
		if idx == 0 || strings.TrimSpace(opt) == "" {
			continue // it's image name
		}
		if isSharedInfra {
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
			return fmt.Errorf("Options not supported: %s", opt)
		}
	}
	return nil
}

func (d *dockerOpts) computeDockerOptsOnVolumeRequirement(isSharedInfra bool, req sdk.Requirement) error {
	// args are separated by a space
	// example: type=bind,source=/hostDir/sourceDir,destination=/dirInJob
	for idx, opt := range strings.Split(req.Value, " ") {
		if isSharedInfra {
			return fmt.Errorf("you could not use this docker options '%s' with a 'shared.infra' hatchery. Please use you own hatchery or remove this option", opt)
		}

		if idx == 0 {
			// it's --mount flag
			if err := d.computeDockerOptsOnVolumeMountRequirement(opt); err != nil {
				return err
			}
		}

	}
	return nil
}

// computeDockerOptsOnVolumeMountRequirement compute Mount struct from value of requirement
func (d *dockerOpts) computeDockerOptsOnVolumeMountRequirement(opt string) error {
	// check that value begin with type= and contains source= / destination=
	if !strings.HasPrefix(opt, "type=") || !strings.Contains(opt, "source=") || !strings.Contains(opt, "destination=") {
		return fmt.Errorf("Invalid mount option. Example:type=bind,source=/hostDir/sourceDir,destination=/dirInJob current:%s", opt)
	}

	var mtype, source, destination, bindPropagation string
	var readonly bool

	// iterate over arg separated by ','
	// type=bind,source=/hostDir/sourceDir,destination=/dirInJob ->
	// [type=bind] [source=/hostDir/sourceDir] [destination=/dirInJob]
	for _, o := range strings.Split(opt, ",") {
		if strings.HasPrefix(o, "type=") {
			mtype = strings.Split(o, "=")[1]
		} else if strings.HasPrefix(o, "source=") {
			source = strings.Split(o, "=")[1]
		} else if strings.HasPrefix(o, "destination=") {
			destination = strings.Split(o, "=")[1]
		} else if strings.HasPrefix(o, "bind-propagation=") {
			bindPropagation = strings.Split(o, "=")[1]
		} else if o == "readonly" {
			readonly = true
		}
	}
	if mtype == "" || source == "" || destination == "" {
		return fmt.Errorf("Invalid mount option - one arg is empty. Example:type=bind,source=/hostDir/sourceDir,destination=/dirInJob current:%s", opt)
	}

	m := mount.Mount{
		Target:   destination,
		Source:   source,
		Type:     mount.Type(mtype),
		ReadOnly: readonly,
	}
	// rprivate is the default value
	// see https://docs.docker.com/engine/admin/volumes/bind-mounts/#choosing-the--v-or-mount-flag
	if bindPropagation != "" {
		m.BindOptions = &mount.BindOptions{Propagation: mount.Propagation(bindPropagation)}
	}

	d.mounts = append(d.mounts, m)

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
	return fmt.Errorf("Wrong format of ports arguments. Example: --port=8081:8182/tcp")
}
