package swarm

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"bytes"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/fsouza/go-dockerclient"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/viper"
)

var hatcherySwarm *HatcherySwarm

//HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	hatch              *sdk.Hatchery
	dockerClient       *docker.Client
	onlyWithServiceReq bool
	maxContainers      int
	defaultMemory      int
	workerTTL          int
}

//Init connect the hatchery to the docker api
func (h *HatcherySwarm) Init() error {
	var err error

	h.dockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		log.Critical("Unable to connect to a docker client")
		return err
	}

	if errPing := h.dockerClient.Ping(); errPing != nil {
		log.Critical("Unable to ping docker host")
		return errPing
	}

	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery"
	}

	name += "-swarm"
	h.hatch = &sdk.Hatchery{
		Name: name,
	}

	if err := hatchery.Register(h.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
		return err
	}

	log.Notice("Swarm Hatchery ready to run !")

	go h.killAwolWorkerRoutine()
	return nil
}

// KillWorker kill the worker
func (h *HatcherySwarm) KillWorker(worker sdk.Worker) error {
	log.Warning("killing container %s", worker.Name)
	containers, err := h.dockerClient.ListContainers(docker.ListContainersOptions{
		All: true,
	})

	if err != nil {
		return err
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(worker.Name, "/", "", 1) {
			//Kill the container, and all linked containers
			h.killAndRemove(containers[i].ID)
		}
	}

	return nil
}

func (h *HatcherySwarm) getContainer(name string) (*docker.APIContainers, error) {
	containers, err := h.dockerClient.ListContainers(docker.ListContainersOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}

func (h *HatcherySwarm) killAndRemove(ID string) error {
	container, err := h.dockerClient.InspectContainer(ID)
	if err != nil {
		return err
	}

	network, err := h.dockerClient.NetworkInfo(container.NetworkSettings.NetworkID)
	if err != nil {
		return err
	}

	if netname, ok := network.Labels["worker_net"]; ok {
		log.Notice("Remove network %s", netname)
		for id := range network.Containers {
			log.Notice("Remove container %s", id)
			if err := h.dockerClient.KillContainer(docker.KillContainerOptions{
				ID:     id,
				Signal: docker.SIGKILL,
			}); err != nil {
				log.Warning("Unable to kill container %s", err)

				if log.IsDebug() {
					buffer := new(bytes.Buffer)

					err := h.dockerClient.Logs(docker.LogsOptions{
						Container:    id,
						Stderr:       true,
						Stdout:       true,
						ErrorStream:  buffer,
						OutputStream: buffer,
					})

					if err != nil {
						log.Warning("Unable to get logs for container %s", id)
					} else {
						log.Debug("***** Container %s logs : ")
						log.Debug("%s", buffer.String())
						log.Debug("**************************")
					}
				}

				continue
			}

			if err := h.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
				ID:            id,
				RemoveVolumes: true,
				Force:         true,
			}); err != nil {
				log.Warning("Unable to remove container %s", err)
			}
		}
	} else {
		log.Notice("Remove container %s", ID)
		if err := h.dockerClient.KillContainer(docker.KillContainerOptions{
			ID:     ID,
			Signal: docker.SIGKILL,
		}); err != nil {
			log.Warning("Unable to kill container %s", err)
		}

		if err := h.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
			ID: ID,
		}); err != nil {
			log.Warning("Unable to remove container %s", err)
		}
	}

	return nil
}

//SpawnWorker start a new docker container
func (h *HatcherySwarm) SpawnWorker(model *sdk.Model, req []sdk.Requirement) error {
	//name is the name of the worker and the name of the container
	name := fmt.Sprintf("swarmy-%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))

	log.Notice("Spawning worker %s with requirements %v", name, req)

	//Create a network
	network := name + "-net"
	h.createNetwork(network)

	//Memory for the worker
	memory := int64(h.defaultMemory)

	for _, r := range req {
		if r.Type == sdk.MemoryRequirement {
			var err error
			memory, err = strconv.ParseInt(r.Value, 10, 64)
			if err != nil {
				log.Warning("SpawnWorker>Unable to parse memory requirement %s :s\n", memory, err)
				return err
			}
		}
	}

	//Prepare worker services from requirements
	services := []string{}
	for _, r := range req {
		if r.Type == sdk.ServiceRequirement {
			//name= <alias> => the name of the host put in /etc/hosts of the worker
			//value= "postgres:latest env_1=blabla env_2=blabla"" => we can add env variables in requirement name
			tuple := strings.Split(r.Value, " ")
			img := tuple[0]
			env := []string{}
			if len(tuple) > 1 {
				env = append(env, tuple[1:]...)
			}
			serviceName := r.Name + "-" + name

			//labels are used to make container cleanup easier. We "link" the service to its worker this way.
			labels := map[string]string{
				"service_worker": name,
				"service_name":   serviceName,
			}
			//Start the services
			if err := h.createAndStartContainer(serviceName, img, network, name, []string{}, env, labels, 0); err != nil {
				log.Warning("SpawnWorker>Unable to start required container: %s\n", err)
				return err
			}
			services = append(services, serviceName)
		}
	}

	//cmd is the command to start the worker (we need curl to download current version of the worker binary)
	cmd := []string{"sh", "-c", fmt.Sprintf("curl %s/download/worker/`uname -m` -o worker && echo chmod worker && chmod +x worker && echo starting worker && ./worker", sdk.Host)}

	//CDS env needed by the worker binary
	env := []string{
		"CDS_API" + "=" + sdk.Host,
		"CDS_NAME" + "=" + name,
		"CDS_KEY" + "=" + viper.GetString("token"),
		"CDS_MODEL" + "=" + strconv.FormatInt(model.ID, 10),
		"CDS_HATCHERY" + "=" + strconv.FormatInt(h.hatch.ID, 10),
		"CDS_TTL" + "=" + strconv.Itoa(h.workerTTL),
	}

	//labels are used to make container cleanup easier
	labels := map[string]string{
		"worker_model":        strconv.FormatInt(model.ID, 10),
		"worker_name":         name,
		"worker_requirements": strings.Join(services, ","),
	}

	//start the worker
	if err := h.createAndStartContainer(name, model.Image, network, "worker", cmd, env, labels, memory); err != nil {
		log.Warning("SpawnWorker> Unable to start container %s\n", err)
	}

	return nil
}

//create the docker bridge
func (h *HatcherySwarm) createNetwork(name string) error {
	log.Debug("createAndStartContainer> Create network %s\n", name)
	_, err := h.dockerClient.CreateNetwork(docker.CreateNetworkOptions{
		Name:           name,
		Driver:         "bridge",
		Internal:       false,
		CheckDuplicate: true,
		EnableIPv6:     false,
		IPAM: docker.IPAMOptions{
			Driver: "default",
		},
		Labels: map[string]string{
			"worker_net": name,
		},
	})
	return err
}

//shortcut to create+start(=run) a container
func (h *HatcherySwarm) createAndStartContainer(name, image, network, networkAlias string, cmd, env []string, labels map[string]string, memory int64) error {
	//Memory is set to 1GB by default
	if memory == 0 {
		memory = 1024
	} else {
		//Moaaaaar memory
		memory = memory * 110 / 100
	}
	log.Notice("createAndStartContainer> Create container %s from %s on network %s as %s (memory=%dMB)", name, image, network, networkAlias, memory)
	opts := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:      image,
			Cmd:        cmd,
			Env:        env,
			Labels:     labels,
			Memory:     memory * 1024 * 1024, //from MB to B
			MemorySwap: -1,
		},
		NetworkingConfig: &docker.NetworkingConfig{
			EndpointsConfig: map[string]*docker.EndpointConfig{
				network: &docker.EndpointConfig{
					Aliases: []string{networkAlias, name},
				},
			},
		},
	}

	c, err := h.dockerClient.CreateContainer(opts)
	if err != nil {
		log.Warning("startAndCreateContainer> Unable to create container %s\n", err)
		return err
	}

	if err := h.dockerClient.StartContainer(c.ID, nil); err != nil {
		log.Warning("startAndCreateContainer> Unable to start container %s\n", err)
		return err
	}
	return nil
}

// CanSpawn checks if the model can be spawned by this hatchery
func (h *HatcherySwarm) CanSpawn(model *sdk.Model, req []sdk.Requirement) bool {
	if model.Type != sdk.Docker {
		return false
	}

	//List all containers to check if we can spawn a new one
	if cs, _ := h.dockerClient.ListContainers(docker.ListContainersOptions{}); len(cs) > h.maxContainers {
		return false
	}

	//Get links from requirements
	var atLeastOneLink bool
	links := map[string]string{}
	for _, r := range req {
		if r.Type == sdk.ServiceRequirement {
			atLeastOneLink = true
			links[r.Name] = strings.Split(r.Value, " ")[0]
		}
	}

	//This hatchery may only manage container with links
	if (!atLeastOneLink) && h.onlyWithServiceReq {
		return false
	}

	log.Notice("CanSpawn> %s need %v", model.Name, links)

	images, err := h.dockerClient.ListImages(docker.ListImagesOptions{})
	if err != nil {
		log.Warning("Unable to get images : %s", err)
	}

	var imageFound bool
	for _, img := range images {
		for _, t := range img.RepoTags {
			if model.Image == t {
				imageFound = true
				goto pullImage
			}
		}
	}

pullImage:
	if !imageFound {
		//Pull the worker image
		opts := docker.PullImageOptions{
			Repository:   model.Image,
			OutputStream: nil,
		}
		auth := docker.AuthConfiguration{}
		log.Notice("CanSpawn> pulling image %s", model.Image)
		if err := h.dockerClient.PullImage(opts, auth); err != nil {
			log.Warning("Unable to pull image %s : %s", model.Image, err)
			return false
		}
	}

	//Pull the service image
	for _, i := range links {
		var imageFound2 bool
		for _, img := range images {
			for _, t := range img.RepoTags {
				if i == t {
					imageFound2 = true
					goto pullLink
				}
			}
		}
	pullLink:
		if !imageFound2 {
			opts := docker.PullImageOptions{
				Repository:   i,
				OutputStream: nil,
			}
			auth := docker.AuthConfiguration{}
			log.Notice("CanSpawn> pulling image %s", i)
			if err := h.dockerClient.PullImage(opts, auth); err != nil {
				log.Warning("Unable to pull image %s : %s", i, err)
				return false
			}
		}
	}

	//Ready to spawn
	return true
}

// WorkerStarted returns the number of started workers
func (h *HatcherySwarm) WorkerStarted(model *sdk.Model) int {
	if model.Type != sdk.Docker {
		return 0
	}

	containers, err := h.dockerClient.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		log.Warning("WorkerStarted> error listing containers : %s", err)
	}

	list := []string{}
	for _, c := range containers {
		log.Info("Container : %s %s [%s]", c.ID, c.Image, c.Status)
		if c.Image == model.Image {
			list = append(list, c.ID)
		}
	}

	log.Notice("WorkerStarted> %s \t %d", model.Name, len(list))

	return len(list)
}

// Hatchery returns Hatchery instances
func (h *HatcherySwarm) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// ID returns ID of the Hatchery
func (h *HatcherySwarm) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

func (h *HatcherySwarm) killAwolWorkerRoutine() {
	for {
		time.Sleep(30 * time.Second)
		h.killAwolWorker()
	}
}

func (h *HatcherySwarm) killAwolWorker() {
	apiworkers, err := sdk.GetWorkers()
	if err != nil {
		log.Warning("Cannot get workers: %s", err)
		os.Exit(1)
	}

	containers, errList := h.dockerClient.ListContainers(docker.ListContainersOptions{
		All: true,
	})
	if errList != nil {
		log.Warning("Cannot list containers: %s", errList)
		os.Exit(1)
	}

	//Checking workers
	oldContainers := []docker.APIContainers{}
	for _, c := range containers {
		//Ignore containers spawned by other things that this Hatchery
		if c.Labels["worker_name"] == "" {
			continue
		}
		//If there isn't any worker registered on the API. Kill the container
		if len(apiworkers) == 0 {
			oldContainers = append(oldContainers, c)
			continue
		}
		//Loop on all worker registered on the API
		//Try to find the worker matching this container
		var found = false
		for _, n := range apiworkers {
			if n.Name == c.Names[0] || n.Name == strings.Replace(c.Names[0], "/", "", 1) {
				found = true
				// If worker is disabled, kill it
				if n.Status == sdk.StatusDisabled {
					log.Info("Worker %s is disabled. Kill it with fire !\n", c.Names[0])
					oldContainers = append(oldContainers, c)
					break
				}
			}
		}
		//If the container doesn't match any worker : Kill it.
		if !found {
			oldContainers = append(oldContainers, c)
		}
	}

	//Delete the workers
	for _, c := range oldContainers {
		h.killAndRemove(c.ID)
		log.Notice("HatcherySwarm.killAwolWorker> Delete worker %s\n", c.Names[0])
	}

	var errLC error
	containers, errLC = h.dockerClient.ListContainers(docker.ListContainersOptions{
		All: true,
	})
	if errLC != nil {
		log.Warning("Cannot get containers: %s", errLC)
		return
	}

	//Checking services
	for _, c := range containers {
		if c.Labels["service_worker"] == "" {
			continue
		}
		if w, _ := h.getContainer(c.Labels["service_worker"]); w == nil {
			oldContainers = append(oldContainers, c)
			continue
		}
	}

	for _, c := range oldContainers {
		h.killAndRemove(c.ID)
		log.Notice("HatcherySwarm.killAwolWorker> Delete worker %s\n", c.Names[0])
	}

	//Checking networks
	nets, errLN := h.dockerClient.ListNetworks()
	if errLN != nil {
		log.Warning("Cannot get networks: %s", errLN)
		return
	}

	for _, n := range nets {
		if n.Driver != "bridge" || n.Name == "docker0" || n.Name == "bridge" {
			continue
		}

		if _, ok := n.Labels["worker_net"]; !ok {
			continue
		}

		if len(n.Containers) > 0 {
			continue
		}

		log.Notice("HatcherySwarm.killAwolWorker> Delete network %s", n.Name)
		if err := h.dockerClient.RemoveNetwork(n.ID); err != nil {
			log.Warning("HatcherySwarm.killAwolWorker> Unable to delete network %s", n.Name)
		}
	}
}
