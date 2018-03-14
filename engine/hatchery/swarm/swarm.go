package swarm

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// New instanciates a new Hatchery Swarm
func New() *HatcherySwarm {
	return new(HatcherySwarm)
}

// Serve start the HatcherySwarm server
func (h *HatcherySwarm) Serve(ctx context.Context) error {
	return hatchery.Create(h)
}

//Init connect the hatchery to the docker api
func (h *HatcherySwarm) Init() error {
	h.hatch = &sdk.Hatchery{
		Name:    h.Configuration().Name,
		Version: sdk.VERSION,
	}

	h.client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		h.hatch.Name,
	)
	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

	var errc error
	h.dockerClient, errc = docker.NewEnvClient()
	if errc != nil {
		log.Error("Unable to connect to a docker client:%s", errc)
		return errc
	}

	if _, errPing := h.dockerClient.Ping(context.Background()); errPing != nil {
		log.Error("Unable to ping docker host:%s", errPing)
		return errPing
	}

	go h.killAwolWorkerRoutine()
	return nil
}

// SpawnWorker start a new docker container
// User can add option on prerequisite, as --port and --privileged
// but only hatchery NOT 'shared.infra' can launch containers with options
func (h *HatcherySwarm) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {
	//name is the name of the worker and the name of the container
	name := fmt.Sprintf("swarmy-%s-%s", strings.ToLower(spawnArgs.Model.Name), strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1))
	if spawnArgs.RegisterOnly {
		name = "register-" + name
	}

	log.Debug("SpawnWorker> Spawning worker %s - %s", name, spawnArgs.LogInfo)

	//Create a network
	network := name + "-net"
	h.createNetwork(network)

	//Memory for the worker
	memory := int64(h.Config.DefaultMemory)

	services := []string{}

	if spawnArgs.JobID > 0 {
		for _, r := range spawnArgs.Requirements {
			if r.Type == sdk.MemoryRequirement {
				var err error
				memory, err = strconv.ParseInt(r.Value, 10, 64)
				if err != nil {
					log.Warning("SpawnWorker>Unable to parse memory requirement %s :s", memory, err)
					return "", err
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
					"hatchery":       h.Config.Name,
				}

				//Start the services
				args := containerArgs{
					name:         serviceName,
					image:        img,
					network:      network,
					networkAlias: r.Name,
					cmd:          []string{},
					env:          env,
					labels:       labels,
					memory:       serviceMemory,
				}

				if err := h.createAndStartContainer(args); err != nil {
					log.Warning("SpawnWorker>Unable to start required container: %s", err)
					return "", err
				}
				services = append(services, serviceName)
			}
		}
	}

	var registerCmd string
	if spawnArgs.RegisterOnly {
		registerCmd = " register"
		memory = hatchery.MemoryRegisterContainer
	}

	//cmd is the command to start the worker (we need curl to download current version of the worker binary)
	cmd := []string{"sh", "-c", fmt.Sprintf("curl %s/download/worker/linux/`uname -m` -o worker && echo chmod worker && chmod +x worker && echo starting worker && ./worker%s", h.Client().APIURL(), registerCmd)}

	//CDS env needed by the worker binary
	env := []string{
		"CDS_API" + "=" + h.Configuration().API.HTTP.URL,
		"CDS_NAME" + "=" + name,
		"CDS_TOKEN" + "=" + h.Configuration().API.Token,
		"CDS_MODEL" + "=" + strconv.FormatInt(spawnArgs.Model.ID, 10),
		"CDS_HATCHERY" + "=" + strconv.FormatInt(h.hatch.ID, 10),
		"CDS_HATCHERY_NAME" + "=" + h.hatch.Name,
		"CDS_TTL" + "=" + strconv.Itoa(h.Config.WorkerTTL),
		"CDS_SINGLE_USE=1",
	}

	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Host != "" {
		env = append(env, "CDS_GRAYLOG_HOST"+"="+h.Configuration().Provision.WorkerLogsOptions.Graylog.Host)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.Port > 0 {
		env = append(env, fmt.Sprintf("CDS_GRAYLOG_PORT=%d", h.Configuration().Provision.WorkerLogsOptions.Graylog.Port))
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey != "" {
		env = append(env, "CDS_GRAYLOG_EXTRA_KEY"+"="+h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey)
	}
	if h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue != "" {
		env = append(env, "CDS_GRAYLOG_EXTRA_VALUE"+"="+h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue)
	}
	if h.Configuration().API.GRPC.URL != "" && spawnArgs.Model.Communication == sdk.GRPC {
		env = append(env, fmt.Sprintf("CDS_GRPC_API=%s", h.Configuration().API.GRPC.URL))
		env = append(env, fmt.Sprintf("CDS_GRPC_INSECURE=%t", h.Configuration().API.GRPC.Insecure))
	}

	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			env = append(env, fmt.Sprintf("CDS_BOOKED_WORKFLOW_JOB_ID=%d", spawnArgs.JobID))
		} else {
			env = append(env, fmt.Sprintf("CDS_BOOKED_PB_JOB_ID=%d", spawnArgs.JobID))
		}
	}

	//labels are used to make container cleanup easier
	labels := map[string]string{
		"worker_model":        strconv.FormatInt(spawnArgs.Model.ID, 10),
		"worker_name":         name,
		"worker_requirements": strings.Join(services, ","),
		"hatchery":            h.Config.Name,
	}

	dockerOpts, errDockerOpts := computeDockerOpts(h.hatch.IsSharedInfra, spawnArgs.Requirements)
	if errDockerOpts != nil {
		return name, errDockerOpts
	}

	args := containerArgs{
		name:         name,
		image:        spawnArgs.Model.Image,
		network:      network,
		networkAlias: "worker",
		cmd:          cmd,
		env:          env,
		labels:       labels,
		memory:       memory,
		dockerOpts:   *dockerOpts,
	}

	//start the worker
	if err := h.createAndStartContainer(args); err != nil {
		log.Warning("SpawnWorker> Unable to start container named %s with image %s err:%s", name, spawnArgs.Model.Image, err)
	}

	return name, nil
}

// ModelType returns type of hatchery
func (*HatcherySwarm) ModelType() string {
	return sdk.Docker
}

const (
	timeoutPullImage = 10 * time.Minute
)

// CanSpawn checks if the model can be spawned by this hatchery
func (h *HatcherySwarm) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	//List all containers to check if we can spawn a new one
	cs, errList := h.getContainers(types.ContainerListOptions{})
	if errList != nil {
		log.Error("CanSpawn> Unable to list containers: %s", errList)
		return false
	}

	//List all workers
	ws, errWList := h.getWorkersStarted(cs)
	if errWList != nil {
		log.Error("CanSpawn> Unable to list workers: %s", errWList)
		return false
	}

	if len(cs) > h.Config.MaxContainers {
		log.Debug("CanSpawn> max containers reached. current:%d max:%d", len(cs), h.Config.MaxContainers)
		return false
	}

	//Get links from requirements
	links := map[string]string{}

	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement {
			links[r.Name] = strings.Split(r.Value, " ")[0]
		}
	}

	// hatcherySwarm.ratioService: Percent reserved for spawning worker with service requirement
	// if no link -> we need to check ratioService
	if len(links) == 0 {
		if h.Config.RatioService >= 100 {
			log.Debug("CanSpawn> ratioService 100 by conf - no spawn worker without CDS Service")
			return false
		}
		if len(cs) > 0 {
			percentFree := 100 - (100 * len(ws) / h.Config.MaxContainers)
			if percentFree <= h.Config.RatioService {
				log.Debug("CanSpawn> ratio reached. percentFree:%d ratioService:%d", percentFree, h.Config.RatioService)
				return false
			}
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

	var images []types.ImageSummary
	// if we don't need to force pull links, we check if model is "latest"
	// if model is not "latest" tag too, ListImages to get images locally
	if listImagesToDoForLinkedImages || !strings.HasSuffix(model.Image, ":latest") {
		var errl error
		images, errl = h.dockerClient.ImageList(context.Background(), types.ImageListOptions{
			All: true,
		})
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
		if err := h.pullImage(model.Image, timeoutPullImage); err != nil {
			//the error is already logged
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
			if err := h.pullImage(i, timeoutPullImage); err != nil {
				//the error is already logged
				return false
			}
		}
	}

	//Ready to spawn
	log.Debug("CanSpawn> %s can be spawned", model.Name)
	return true
}

func (h *HatcherySwarm) getWorkersStarted(containers []types.Container) ([]types.Container, error) {
	if containers == nil {
		var errList error
		// get only started containers
		containers, errList = h.getContainers(types.ContainerListOptions{})
		if errList != nil {
			log.Error("WorkersStarted> Unable to list containers: %s", errList)
			return nil, errList
		}
	}

	res := []types.Container{}
	//We only count worker
	for _, c := range containers {
		cont, err := h.getContainer(c.Names[0], types.ContainerListOptions{})
		if err != nil {
			log.Error("WorkersStarted> Unable to get worker %s: %v", c.Names[0], err)
			continue
		}
		if _, ok := cont.Labels["worker_name"]; ok {
			if hatch, ok := cont.Labels["hatchery"]; !ok || hatch == h.Config.Name {
				res = append(res, *cont)
			}
		}
	}
	return res, nil
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcherySwarm) WorkersStarted() int {
	workers, _ := h.getWorkersStarted(nil)
	return len(workers)
}

// WorkersStartedByModel returns the number of started workers
func (h *HatcherySwarm) WorkersStartedByModel(model *sdk.Model) int {
	workers, errList := h.getWorkersStarted(nil)
	if errList != nil {
		log.Error("WorkersStartedByModel> Unable to list containers: %s", errList)
		return 0
	}

	list := []string{}
	for _, c := range workers {
		log.Debug("Container : %s %s [%s]", c.ID, c.Image, c.Status)
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

//Client returns cdsclient instance
func (h *HatcherySwarm) Client() cdsclient.Interface {
	return h.client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcherySwarm) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
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
		time.Sleep(10 * time.Second)
		h.killAwolWorker()
	}
}

func (h *HatcherySwarm) listAwolWorkers() ([]types.Container, error) {
	apiworkers, err := h.Client().WorkerList()
	if err != nil {
		return nil, sdk.WrapError(err, "listAwolWorkers> Cannot get workers")
	}

	containers, errList := h.getWorkersStarted(nil)
	if errList != nil {
		return nil, sdk.WrapError(err, "listAwolWorkers> Cannot list containers")
	}

	//Checking workers
	oldContainers := []types.Container{}
	for _, c := range containers {
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
					log.Debug("listAwolWorkers> Worker %s is disabled. Kill it with fire !", c.Names[0])
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

	return oldContainers, nil
}

func (h *HatcherySwarm) killAwolWorker() error {
	oldContainers, err := h.listAwolWorkers()
	if err != nil {
		log.Warning("killAwolWorker> Cannot list workers %s", err)
		return err
	}

	//Delete the workers
	for _, c := range oldContainers {
		log.Debug("killAwolWorker> Delete worker %s", c.Names[0])
		if err := h.killAndRemove(c.ID); err != nil {
			log.Error("killAwolWorker> %v", err)
		}
	}

	containers, errC := h.getContainers(types.ContainerListOptions{All: true})
	if errC != nil {
		log.Warning("killAwolWorker> Cannot list containers: %s", errC)
		return errC
	}

	//Checking services
	for _, c := range containers {
		if c.Labels["service_worker"] == "" {
			continue
		}
		//check if the service is linked to a worker which doesn't exist
		if w, _ := h.getContainer(c.Labels["service_worker"], types.ContainerListOptions{All: true}); w == nil {
			oldContainers = append(oldContainers, c)
			continue
		}
	}

	for _, c := range oldContainers {
		log.Debug("killAwolWorker> Delete worker %s", c.Names[0])
		if err := h.killAndRemove(c.ID); err != nil {
			log.Error("killAwolWorker> %v", err)
		}
	}

	return h.killAwolNetworks()
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcherySwarm) NeedRegistration(m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
