package swarm

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/gorilla/mux"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// New instanciates a new Hatchery Swarm
func New() *HatcherySwarm {
	s := new(HatcherySwarm)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

//Init connect the hatchery to the docker api
func (h *HatcherySwarm) Init() error {
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

	go h.routines(context.Background())

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

	//Memory for the worker
	memory := int64(h.Config.DefaultMemory)

	if spawnArgs.Model.ModelDocker.Memory != 0 {
		memory = spawnArgs.Model.ModelDocker.Memory
	}

	var network, networkAlias string
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
				//Create a network if not already created
				if network == "" {
					network = name + "-net"
					networkAlias = "worker"
					if err := h.createNetwork(network); err != nil {
						log.Warning("SpawnWorker>Unable to create network %s for jobID %d : %v", network, spawnArgs.JobID, err)
						continue
					}
				}

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

				if spawnArgs.IsWorkflowJob {
					labels["service_job_id"] = fmt.Sprintf("%d", spawnArgs.JobID)
					labels["service_id"] = fmt.Sprintf("%d", r.ID)
					labels["service_req_name"] = r.Name
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
					entryPoint:   nil,
				}

				if err := h.createAndStartContainer(args, spawnArgs); err != nil {
					log.Warning("SpawnWorker>Unable to start required container: %s", err)
					return "", err
				}
				services = append(services, serviceName)
			}
		}
	}

	if spawnArgs.RegisterOnly {
		spawnArgs.Model.ModelDocker.Cmd += " register"
		memory = hatchery.MemoryRegisterContainer
	}

	//labels are used to make container cleanup easier
	labels := map[string]string{
		"worker_model":        strconv.FormatInt(spawnArgs.Model.ID, 10),
		"worker_name":         name,
		"worker_requirements": strings.Join(services, ","),
		"hatchery":            h.Config.Name,
	}

	dockerOpts, errDockerOpts := h.computeDockerOpts(spawnArgs.Requirements)
	if errDockerOpts != nil {
		return name, errDockerOpts
	}

	udataParam := sdk.WorkerArgs{
		API:               h.Configuration().API.HTTP.URL,
		Token:             h.Configuration().API.Token,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              name,
		Model:             spawnArgs.Model.ID,
		TTL:               h.Config.WorkerTTL,
		Hatchery:          h.hatch.ID,
		HatcheryName:      h.hatch.Name,
		GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
		GrpcAPI:           h.Configuration().API.GRPC.URL,
		GrpcInsecure:      h.Configuration().API.GRPC.Insecure,
	}

	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			udataParam.WorkflowJobID = spawnArgs.JobID
		} else {
			udataParam.PipelineBuildJobID = spawnArgs.JobID
		}
	}

	tmpl, errt := template.New("cmd").Parse(spawnArgs.Model.ModelDocker.Cmd)
	if errt != nil {
		return "", errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return "", errTmpl
	}
	cmds := strings.Fields(spawnArgs.Model.ModelDocker.Shell)
	cmds = append(cmds, buffer.String())

	if spawnArgs.Model.ModelDocker.Envs == nil {
		spawnArgs.Model.ModelDocker.Envs = map[string]string{}
	}

	envsWm, errEnv := sdk.TemplateEnvs(udataParam, spawnArgs.Model.ModelDocker.Envs)
	if errEnv != nil {
		return "", errEnv
	}

	envsWm["CDS_FORCE_EXIT"] = "1"
	envsWm["CDS_API"] = udataParam.API
	envsWm["CDS_TOKEN"] = udataParam.Token
	envsWm["CDS_NAME"] = udataParam.Name
	envsWm["CDS_MODEL"] = fmt.Sprintf("%d", udataParam.Model)
	envsWm["CDS_HATCHERY"] = fmt.Sprintf("%d", udataParam.Hatchery)
	envsWm["CDS_HATCHERY_NAME"] = udataParam.HatcheryName
	envsWm["CDS_FROM_WORKER_IMAGE"] = fmt.Sprintf("%v", udataParam.FromWorkerImage)
	envsWm["CDS_INSECURE"] = fmt.Sprintf("%v", udataParam.HTTPInsecure)

	if spawnArgs.JobID > 0 {
		if spawnArgs.IsWorkflowJob {
			envsWm["CDS_BOOKED_WORKFLOW_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		} else {
			envsWm["CDS_BOOKED_PB_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
		}
	}

	if udataParam.GrpcAPI != "" && spawnArgs.Model.Communication == sdk.GRPC {
		envsWm["CDS_GRPC_API"] = udataParam.GrpcAPI
		envsWm["CDS_GRPC_INSECURE"] = fmt.Sprintf("%v", udataParam.GrpcInsecure)
	}

	envs := make([]string, len(envsWm))
	i := 0
	for envName, envValue := range envsWm {
		envs[i] = envName + "=" + envValue
		i++
	}

	args := containerArgs{
		name:         name,
		image:        spawnArgs.Model.ModelDocker.Image,
		network:      network,
		networkAlias: networkAlias,
		cmd:          cmds,
		labels:       labels,
		memory:       memory,
		dockerOpts:   *dockerOpts,
		entryPoint:   []string{},
		env:          envs,
	}

	//start the worker
	if err := h.createAndStartContainer(args, spawnArgs); err != nil {
		log.Warning("SpawnWorker> Unable to start container named %s with image %s err:%s", name, spawnArgs.Model.ModelDocker.Image, err)
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
	cs, errList := h.getContainers(types.ContainerListOptions{All: true})
	if errList != nil {
		log.Error("CanSpawn> Unable to list containers: %s", errList)
		return false
	}

	//List all workers
	ws, errWList := h.getWorkerContainers(cs, types.ContainerListOptions{})
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
	if listImagesToDoForLinkedImages || !strings.HasSuffix(model.ModelDocker.Image, ":latest") {
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
	if !strings.HasSuffix(model.ModelDocker.Image, ":latest") {
	checkImage:
		for _, img := range images {
			for _, t := range img.RepoTags {
				if model.ModelDocker.Image == t {
					imageFound = true
					break checkImage
				}
			}
		}
	}

	if !imageFound {
		if err := h.pullImage(model.ModelDocker.Image, timeoutPullImage); err != nil {
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

func (h *HatcherySwarm) getWorkerContainers(containers []types.Container, option types.ContainerListOptions) ([]types.Container, error) {
	if containers == nil {
		var errList error
		// get only started containers
		containers, errList = h.getContainers(option)
		if errList != nil {
			log.Error("WorkersStarted> Unable to list containers: %s", errList)
			return nil, errList
		}
	}

	res := []types.Container{}
	//We only count worker
	for _, c := range containers {
		cont, err := h.getContainer(c.Names[0], option)
		if err != nil {
			log.Error("WorkersStarted> Unable to get worker %s: %v", c.Names[0], err)
			continue
		}
		// the container could be nil
		if cont == nil {
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
	workers, _ := h.getWorkerContainers(nil, types.ContainerListOptions{})
	return len(workers)
}

// WorkersStartedByModel returns the number of started workers
func (h *HatcherySwarm) WorkersStartedByModel(model *sdk.Model) int {
	workers, errList := h.getWorkerContainers(nil, types.ContainerListOptions{})
	if errList != nil {
		log.Error("WorkersStartedByModel> Unable to list containers: %s", errList)
		return 0
	}

	list := []string{}
	for _, c := range workers {
		log.Debug("Container : %s %s [%s]", c.ID, c.Image, c.Status)
		if c.Image == model.ModelDocker.Image {
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

// Serve start the hatchery server
func (h *HatcherySwarm) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
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

func (h *HatcherySwarm) routines(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go func() {
				err := h.getServicesLogs()
				if err != nil {
					log.Error("Hatchery> swarm> Cannot get service logs : %v", err)
				}
			}()
			go func() {
				_ = h.killAwolWorker()
			}()
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error("Hatchery> Swarm> Exiting routines")
			}
			return
		}
	}
}

func (h *HatcherySwarm) listAwolWorkers() ([]types.Container, error) {
	apiworkers, err := h.CDSClient().WorkerList()
	if err != nil {
		return nil, sdk.WrapError(err, "listAwolWorkers> Cannot get workers")
	}

	containers, errList := h.getWorkerContainers(nil, types.ContainerListOptions{All: true})
	if errList != nil {
		return nil, sdk.WrapError(err, "listAwolWorkers> Cannot list containers")
	}

	//Checking workers
	oldContainers := []types.Container{}
	for _, c := range containers {
		if time.Now().Add(-1*time.Minute).Unix() < c.Created {
			log.Debug("listAwolWorkers> container %s is too young", c.Names[0])
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

	log.Debug("listAwolWorkers> oldContainers: %d", len(oldContainers))
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
			log.Debug("killAwolWorker> Delete worker (service) %s", c.Names[0])
			if err := h.killAndRemove(c.ID); err != nil {
				log.Error("killAwolWorker> service %v", err)
			}
			continue
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
