package swarm

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
	hatch         *sdk.Hatchery
	dockerClient  *docker.Client
	ratioService  int
	maxContainers int
	defaultMemory int
	workerTTL     int
}

//Init connect the hatchery to the docker api
func (h *HatcherySwarm) Init() error {
	var errc error
	h.dockerClient, errc = docker.NewClientFromEnv()
	if errc != nil {
		log.Critical("Unable to connect to a docker client:%s", errc)
		return errc
	}

	if errPing := h.dockerClient.Ping(); errPing != nil {
		log.Critical("Unable to ping docker host:%s", errPing)
		return errPing
	}

	h.hatch = &sdk.Hatchery{
		Name: hatchery.GenerateName("swarm", viper.GetString("name")),
	}

	if err := hatchery.Register(h.hatch, viper.GetString("token")); err != nil {
		log.Warning("Cannot register hatchery: %s", err)
		return err
	}

	log.Notice("Swarm Hatchery ready to run !")

	go h.killAwolWorkerRoutine()
	return nil
}

// KillWorker kill the worker
func (h *HatcherySwarm) KillWorker(worker sdk.Worker) error {
	log.Warning("killing container %s", worker.Name)
	containers, errC := h.getContainers()
	if errC != nil {
		return sdk.WrapError(errC, "KillWorker> Cannot list containers")
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(worker.Name, "/", "", 1) {
			//Kill the container, and all linked containers
			h.killAndRemove(containers[i].ID)
		}
	}

	return nil
}

//This a embeded cache for containers list
var containersCache = struct {
	mu   sync.RWMutex
	list []docker.APIContainers
}{
	mu:   sync.RWMutex{},
	list: []docker.APIContainers{},
}

func (h *HatcherySwarm) getContainers() ([]docker.APIContainers, error) {
	t := time.Now()

	defer log.Debug("getContainers() : %d s", time.Since(t).Seconds())

	containersCache.mu.RLock()
	nbServers := len(containersCache.list)
	containersCache.mu.RUnlock()

	if nbServers == 0 {
		s, err := h.dockerClient.ListContainers(docker.ListContainersOptions{
			All: true,
		})
		if err != nil {
			return nil, sdk.WrapError(err, "getContainers> error: %s")
		}
		containersCache.mu.Lock()
		containersCache.list = s
		containersCache.mu.Unlock()
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			containersCache.mu.Lock()
			containersCache.list = []docker.APIContainers{}
			containersCache.mu.Unlock()
		}()
	}

	return containersCache.list, nil
}

func (h *HatcherySwarm) getContainer(name string) (*docker.APIContainers, error) {
	containers, err := h.getContainers()
	if err != nil {
		return nil, sdk.WrapError(err, "getContainer> cannot getContainers")
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}

func (h *HatcherySwarm) killAndRemove(ID string) error {

	/*

		container, err := h.dockerClient.InspectContainer(ID)
		if err != nil {
			return sdk.WrapError(err, "killAndRemove> cannot InspectContainer")
		}

			network, err := h.dockerClient.NetworkInfo(container.NetworkSettings.NetworkID)
			if err != nil {
				return sdk.WrapError(err, "killAndRemove> cannot NetworkInfo")
			}

			if netname, ok := network.Labels["worker_net"]; ok {
				log.Notice("killAndRemove> Remove network %s", netname)
				for id := range network.Containers {
					log.Notice("killAndRemove> Remove container %s", id)
					if err := h.dockerClient.KillContainer(docker.KillContainerOptions{
						ID:     id,
						Signal: docker.SIGKILL,
					}); err != nil {
						log.Info("killAndRemove> Unable to kill container %s", err)
						continue
					}

					if err := h.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
						ID:            id,
						RemoveVolumes: true,
						Force:         true,
					}); err != nil {
						log.Warning("killAndRemove> Unable to remove container %s", err)
					}
				}
			} else {
	*/
	log.Notice("killAndRemove>Remove container %s", ID)
	if err := h.dockerClient.KillContainer(docker.KillContainerOptions{
		ID:     ID,
		Signal: docker.SIGKILL,
	}); err != nil {
		log.Warning("killAndRemove> Unable to kill container %s", err)
	}

	if err := h.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID: ID,
	}); err != nil {
		log.Warning("killAndRemove> Unable to remove container %s", err)
	}

	return nil
}

//SpawnWorker start a new docker container
func (h *HatcherySwarm) SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob) error {
	//name is the name of the worker and the name of the container
	name := fmt.Sprintf("swarmy-%s-%s", strings.ToLower(model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))

	log.Notice("Spawning worker %s", name)

	//Create a network
	network := name + "-net"
	h.createNetwork(network)

	//Memory for the worker
	memory := int64(h.defaultMemory)

	services := []string{}

	if job != nil {
		for _, r := range job.Job.Action.Requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.ParseInt(r.Value, 10, 64)
				if err != nil {
					log.Warning("SpawnWorker>Unable to parse memory requirement %s :s", memory, err)
					return err
				}
			} else if r.Type == sdk.ServiceRequirement {
				//name= <alias> => the name of the host put in /etc/hosts of the worker
				//value= "postgres:latest env_1=blabla env_2=blabla"" => we can add env variables in requirement name
				tuple := strings.Split(r.Value, " ")
				img := tuple[0]
				env := []string{}
				serviceMemory := int64(1024)
				if len(tuple) > 1 {
					env = append(env, tuple[1:]...)
				}
				//option for power user : set the service memory with CDS_SERVICE_MEMORY=1024
				for _, e := range env {
					if strings.HasPrefix(e, "CDS_SERVICE_MEMORY=") {
						m := strings.Replace(e, "CDS_SERVICE_MEMORY=", "", -1)
						i, err := strconv.Atoi(m)
						if err != nil {
							log.Warning("SpawnWorker> Unable to parse service option %s : %s", e, err)
							continue
						}
						serviceMemory = int64(i)
					}
				}
				serviceName := r.Name + "-" + name

				//labels are used to make container cleanup easier. We "link" the service to its worker this way.
				labels := map[string]string{
					"service_worker": name,
					"service_name":   serviceName,
				}
				//Start the services
				if err := h.createAndStartContainer(serviceName, img, network, r.Name, []string{}, env, labels, serviceMemory); err != nil {
					log.Warning("SpawnWorker>Unable to start required container: %s", err)
					return err
				}
				services = append(services, serviceName)
			}
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
		"CDS_SINGLE_USE=1",
	}

	if job != nil {
		env = append(env, "CDS_BOOKED_JOB_ID"+"="+strconv.FormatInt(job.ID, 10))
	}

	//labels are used to make container cleanup easier
	labels := map[string]string{
		"worker_model":        strconv.FormatInt(model.ID, 10),
		"worker_name":         name,
		"worker_requirements": strings.Join(services, ","),
	}

	//start the worker
	if err := h.createAndStartContainer(name, model.Image, network, "worker", cmd, env, labels, memory); err != nil {
		log.Warning("SpawnWorker> Unable to start container %s", err)
	}

	return nil
}

//create the docker bridge
func (h *HatcherySwarm) createNetwork(name string) error {
	log.Debug("createAndStartContainer> Create network %s", name)
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
	if memory <= 4 {
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
		log.Warning("startAndCreateContainer> Unable to create container with opts: %+v err:%s", opts, err)
		return err
	}

	if err := h.dockerClient.StartContainer(c.ID, nil); err != nil {
		log.Warning("startAndCreateContainer> Unable to start container %s err:%s", c.ID, err)
		return err
	}
	return nil
}

// ModelType returns type of hatchery
func (*HatcherySwarm) ModelType() string {
	return sdk.Docker
}

// CanSpawn checks if the model can be spawned by this hatchery
func (h *HatcherySwarm) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	//List all containers to check if we can spawn a new one
	cs, errList := h.getContainers()
	if errList != nil {
		log.Critical("CanSpawn> Unable to list containers: %s", errList)
		return false
	}

	if len(cs) > h.maxContainers {
		log.Warning("CanSpawn> max containers reached. current:%d max:%d", len(cs), h.maxContainers)
		return false
	}

	//Get links from requirements
	links := map[string]string{}

	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement {
			links[r.Name] = strings.Split(r.Value, " ")[0]
		}
	}

	// hatcherySwarm.ratioService: Percent reserved for spwaning worker with service requirement
	// if no link -> we need to check ratioService
	if len(links) == 0 && len(cs) > 0 {
		percentFree := 100 - (100 * len(cs) / h.maxContainers)
		if percentFree <= hatcherySwarm.ratioService {
			log.Notice("CanSpawn> ratio reached. percentFree:%d ratioService:%d", percentFree, hatcherySwarm.ratioService)
			return false
		}
	}

	log.Debug("CanSpawn> %s need %v", model.Name, links)

	// If one image have a "latest" tag, we don't have to listImage
	listImagesToDoForLinkedImages := true
	for _, i := range links {
		if strings.HasSuffix(i, ":latest") {
			listImagesToDoForLinkedImages = false
			break
		}
	}

	var images []docker.APIImages
	// if we don't need to force pull links, we check if model is "latest"
	// if model is not "latest" tag too, ListImages to get images locally
	if listImagesToDoForLinkedImages || !strings.HasSuffix(model.Image, ":latest") {
		var errl error
		images, errl = h.dockerClient.ListImages(docker.ListImagesOptions{})
		if errl != nil {
			log.Warning("CanSpawn> Unable to list images: %s", errl)
		}
	}

	var imageFound bool

	// model is not latest, check if image exists locally
	if !strings.HasSuffix(model.Image, ":latest") {
	checkImage:
		for _, img := range images {
			for _, t := range img.RepoTags {
				if model.Image == t {
					imageFound = true
					break checkImage
				}
			}
		}
	}

	if !imageFound {
		//Pull the worker image
		opts := docker.PullImageOptions{
			Repository:   model.Image,
			OutputStream: nil,
		}
		auth := docker.AuthConfiguration{}
		log.Notice("CanSpawn> pulling image %s", model.Image)
		if err := h.dockerClient.PullImage(opts, auth); err != nil {
			log.Warning("CanSpawn> Unable to pull image %s : %s", model.Image, err)
			return false
		}
	}

	//Pull the service image
	for _, i := range links {
		var imageFound2 bool

		// model is not latest for this link, check if image exists locally
		if !strings.HasSuffix(i, ":latest") {
		checkLink:
			for _, img := range images {
				for _, t := range img.RepoTags {
					if i == t {
						imageFound2 = true
						break checkLink
					}
				}
			}
		}

		if !imageFound2 {
			opts := docker.PullImageOptions{
				Repository:   i,
				OutputStream: nil,
			}
			auth := docker.AuthConfiguration{}
			log.Notice("CanSpawn> pulling image %s", i)
			if err := h.dockerClient.PullImage(opts, auth); err != nil {
				log.Warning("CanSpawn> Unable to pull image %s : %s", i, err)
				return false
			}
		}
	}

	//Ready to spawn
	log.Debug("CanSpawn> %s can be spawned", model.Name)
	return true
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcherySwarm) WorkersStarted() int {
	containers, errList := h.getContainers()
	if errList != nil {
		log.Critical("WorkersStarted> Unable to list containers: %s", errList)
		return 0
	}
	return len(containers)
}

// WorkersStartedByModel returns the number of started workers
func (h *HatcherySwarm) WorkersStartedByModel(model *sdk.Model) int {
	containers, errList := h.getContainers()
	if errList != nil {
		log.Critical("WorkersStartedByModel> Unable to list containers: %s", errList)
		return 0
	}

	list := []string{}
	for _, c := range containers {
		log.Info("Container : %s %s [%s]", c.ID, c.Image, c.Status)
		if c.Image == model.Image {
			list = append(list, c.ID)
		}
	}

	log.Debug("WorkersStartedByModel> %s \t %d", model.Name, len(list))
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
		log.Warning("killAwolWorker> Cannot get workers: %s", err)
		os.Exit(1)
	}

	containers, errList := h.getContainers()
	if errList != nil {
		log.Warning("killAwolWorker> Cannot list containers: %s", errList)
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
					log.Info("killAwolWorker> Worker %s is disabled. Kill it with fire !", c.Names[0])
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
		log.Notice("killAwolWorker> Delete worker %s", c.Names[0])
		if err := h.killAndRemove(c.ID); err != nil {
			log.Warning("killAwolWorker> Cannot killAndRemove worker id: %s, err:%s", c.ID, err)
		}
	}

	var errC error
	containers, errC = h.getContainers()
	if errC != nil {
		log.Warning("killAwolWorker> Cannot list containers: %s", errC)
		os.Exit(1)
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
		log.Notice("killAwolWorker> Delete worker %s", c.Names[0])
	}

	//Checking networks
	nets, errLN := h.dockerClient.ListNetworks()
	if errLN != nil {
		log.Warning("killAwolWorker> Cannot get networks: %s", errLN)
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

		log.Notice("killAwolWorker> Delete network %s", n.Name)
		if err := h.dockerClient.RemoveNetwork(n.ID); err != nil {
			log.Warning("killAwolWorker> Unable to delete network %s", n.Name)
		}
	}
}
