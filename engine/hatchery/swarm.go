package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/fsouza/go-dockerclient"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	sync               sync.Mutex
	hatch              *hatchery.Hatchery
	dockerClient       *docker.Client
	onlyWithServiceReq bool
	maxContainers      int
}

//ParseConfig do nothing
func (h *HatcherySwarm) ParseConfig() {}

//Init connect the hatchery to the docker api
func (h *HatcherySwarm) Init() error {
	var err error

	if os.Getenv("DOCKER_HOST") == "" {
		return errors.New("Please export docker client env variables DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH")
	}

	if os.Getenv("ONLY_WITH_SERVICE_REQ") == "true" {
		h.onlyWithServiceReq = true
	}

	if os.Getenv("MAX_CONTAINER") == "" {
		h.maxContainers = 10
	} else {
		h.maxContainers, err = strconv.Atoi(os.Getenv("MAX_CONTAINER"))
		if err != nil {
			log.Critical("Invalid MAX_CONTAINER")
			return err
		}
	}

	h.dockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		log.Critical("Unable to connect to a docker client")
		return err
	}

	if err := h.dockerClient.Ping(); err != nil {
		log.Critical("Unable to ping docker host")
		return err
	}

	// Register without declaring model
	name, err := os.Hostname()
	if err != nil {
		log.Warning("Cannot retrieve hostname: %s\n", err)
		name = "cds-hatchery"
	}
	name += "-swarm"
	h.hatch = &hatchery.Hatchery{
		Name: name,
	}

	if err := register(h.hatch); err != nil {
		log.Warning("Cannot register hatchery: %s\n", err)
		return err
	}

	log.Notice("Swarm Hatchery ready to run !")

	go h.killAwolWorker()
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
	log.Debug("Remove container %s", ID)
	container, err := h.dockerClient.InspectContainer(ID)
	if err != nil {
		return err
	}

	links := container.HostConfig.Links
	for _, l := range links {
		log.Debug("Remove linked containers : %s", l)
		if strings.Contains(l, ":") {
			c, _ := h.getContainer(strings.Split(l, ":")[0])
			if c != nil {
				defer h.killAndRemove(c.ID)
			}
		} else {
			log.Warning("I dont know what to do with %s", l)
		}
	}

	h.dockerClient.KillContainer(docker.KillContainerOptions{
		ID:     ID,
		Signal: docker.SIGKILL,
	})

	err = h.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: ID,
	})

	return err
}

//SpawnWorker start a new docker container
func (h *HatcherySwarm) SpawnWorker(model *sdk.Model, req []sdk.Requirement) error {

	//name is the name of the worker and the name of the container
	name := fmt.Sprintf("%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))

	log.Debug("Spawning worker %s with requirements %v", name, req)

	//Create a network
	network := name + "-net"
	h.createNetwork(network)

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
			if err := h.createAndStartContainer(serviceName, img, network, name, []string{}, env, labels); err != nil {
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
		"CDS_KEY" + "=" + uk,
		"CDS_MODEL" + "=" + strconv.FormatInt(model.ID, 10),
		"CDS_HATCHERY" + "=" + strconv.FormatInt(h.hatch.ID, 10),
	}

	//labels are used to make container cleanup easier
	labels := map[string]string{
		"worker_model":        strconv.FormatInt(model.ID, 10),
		"worker_name":         name,
		"worker_requirements": strings.Join(services, ","),
	}

	//start the worker
	if err := h.createAndStartContainer(name, model.Image, network, "worker", cmd, env, labels); err != nil {
		log.Warning("SpawnWorker> Unable to start container %s\n", err)
	}

	return nil
}

func (h *HatcherySwarm) createNetwork(name string) error {
	_, err := h.dockerClient.CreateNetwork(docker.CreateNetworkOptions{
		Name:           name,
		Driver:         "bridge",
		Internal:       false,
		CheckDuplicate: true,
		EnableIPv6:     false,
		IPAM: docker.IPAMOptions{
			Driver: "default",
		},
	})
	return err
}

//shortcut to create+start(=run) a container
func (h *HatcherySwarm) createAndStartContainer(name, image, network, networkAlias string, cmd, env []string, labels map[string]string) error {
	log.Debug("createAndStartContainer> Create container %s from %s\n", name, image)
	opts := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:  image,
			Cmd:    cmd,
			Env:    env,
			Labels: labels,
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
	log.Debug("CanSpawn> Checking %s %v", model.Name, req)
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

	//Pull the service image
	for _, i := range links {
		opts := docker.PullImageOptions{
			Repository:   i,
			OutputStream: nil,
		}
		log.Notice("CanSpawn> pulling image %s", i)
		if err := h.dockerClient.PullImage(opts, auth); err != nil {
			log.Warning("Unable to pull image %s : %s", i, err)
			return false
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

// SetWorkerModelID does nothing
func (h *HatcherySwarm) SetWorkerModelID(int64) {}

// Hatchery returns Hatchery instances
func (h *HatcherySwarm) Hatchery() *hatchery.Hatchery {
	return h.hatch
}

// ID returns ID of the Hatchery
func (h *HatcherySwarm) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Mode returns DockerAPIMode value
func (h *HatcherySwarm) Mode() string {
	return SwarmMode
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
		return
	}

	containers, err := h.dockerClient.ListContainers(docker.ListContainersOptions{
		All: true,
	})

	oldContainers := []docker.APIContainers{}
	//Checking workers
	for _, c := range containers {
		if c.Labels["worker_name"] == "" {
			continue
		}
		var found bool
		for _, n := range apiworkers {
			// If worker is disabled, kill it
			if n.Name == c.Names[0] {
				if n.Status == sdk.StatusDisabled {
					log.Info("Worker %s is disabled. Kill it with fire !\n", c.Names[0])
					oldContainers = append(oldContainers, c)
					continue
				}
				found = true
			}
		}
		if !found {
			oldContainers = append(oldContainers, c)
		}
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

}
