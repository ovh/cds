package swarm

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// New instanciates a new Hatchery Swarm
func New() *HatcherySwarm {
	s := new(HatcherySwarm)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

var _ hatchery.InterfaceWithModels = new(HatcherySwarm)

// InitHatchery connect the hatchery to the docker api
func (h *HatcherySwarm) InitHatchery(ctx context.Context) error {
	h.dockerClients = map[string]*dockerClient{}

	if len(h.Config.DockerEngines) == 0 {
		d, errc := docker.NewClientWithOpts(docker.FromEnv)
		if errc != nil {
			log.Error(ctx, "hatchery> swarm> Please export docker client env variables DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH")
			log.Error(ctx, "hatchery> swarm> unable to connect to a docker client:%s", errc)
			return errc
		}
		ctxDocker, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, errPing := d.Ping(ctxDocker); errPing != nil {
			log.Error(ctx, "hatchery> swarm> unable to ping docker host:%s", errPing)
			return errPing
		}
		h.dockerClients["default"] = &dockerClient{
			Client:        *d,
			MaxContainers: h.Config.MaxContainers,
			name:          "default",
		}
		log.Info(ctx, "hatchery> swarm> connected to default docker engine")

	} else {
		for hostName, cfg := range h.Config.DockerEngines {
			log.Info(ctx, "hatchery> swarm> connecting to %s: %s", hostName, cfg.Host)
			httpClient := new(http.Client)
			// max time for a docker pull, but for most of docker request, there is a request with
			// a lower timeout, using context.WithTimeout
			httpClient.Timeout = 10 * time.Minute
			var tlsc *tls.Config
			if cfg.CertPath != "" {
				options := tlsconfig.Options{
					CAFile:             filepath.Join(cfg.CertPath, "ca.pem"),
					CertFile:           filepath.Join(cfg.CertPath, "cert.pem"),
					KeyFile:            filepath.Join(cfg.CertPath, "key.pem"),
					InsecureSkipVerify: cfg.InsecureSkipTLSVerify,
				}
				var err error
				tlsc, err = tlsconfig.Client(options)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error (CertPath=%s): %v", cfg.CertPath, err)
					continue
				}
			} else if cfg.TLSCAPEM != "" && cfg.TLSCERTPEM != "" && cfg.TLSKEYPEM != "" {
				tempDir, err := ioutil.TempDir("", "cert-"+hostName)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error: unable to create temp dir: %v", err)
					continue
				}
				if err := ioutil.WriteFile(filepath.Join(tempDir, "ca.pem"), []byte(cfg.TLSCAPEM), os.FileMode(0600)); err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error: unable to create ca.pem: %v", err)
					continue
				}
				if err := ioutil.WriteFile(filepath.Join(tempDir, "cert.pem"), []byte(cfg.TLSCERTPEM), os.FileMode(0600)); err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error: unable to create cert.pem: %v", err)
					continue
				}
				if err := ioutil.WriteFile(filepath.Join(tempDir, "key.pem"), []byte(cfg.TLSKEYPEM), os.FileMode(0600)); err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error: unable to create key.pem:  %v", err)
					continue
				}
				options := tlsconfig.Options{
					CAFile:             filepath.Join(tempDir, "ca.pem"),
					CertFile:           filepath.Join(tempDir, "cert.pem"),
					KeyFile:            filepath.Join(tempDir, "key.pem"),
					InsecureSkipVerify: cfg.InsecureSkipTLSVerify,
				}
				tlsc, err = tlsconfig.Client(options)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> docker client error: unable to set tlsconfig: %v", err)
					continue
				}
			}

			if tlsc != nil {
				httpClient.Transport = &http.Transport{
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 0 * time.Second,
						DualStack: true,
					}).DialContext,
					MaxIdleConns:          100,
					IdleConnTimeout:       20 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ResponseHeaderTimeout: 30 * time.Second,
					TLSClientConfig:       tlsc,
				}
			} else {
				httpClient.Transport = &http.Transport{
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 0 * time.Second,
						DualStack: true,
					}).DialContext,
					MaxIdleConns:          100,
					IdleConnTimeout:       20 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					ResponseHeaderTimeout: 30 * time.Second,
				}
			}

			var opts = []func(*docker.Client) error{docker.WithHost(cfg.Host), docker.WithVersion(cfg.APIVersion), docker.WithHTTPClient(httpClient)}
			if strings.HasPrefix(cfg.Host, "unix:///") {
				opts = []func(*docker.Client) error{docker.WithHost(cfg.Host)}
			}

			d, errc := docker.NewClientWithOpts(opts...)
			if errc != nil {
				log.Error(ctx, "hatchery> swarm> unable to connect to a docker client:%s for host %s (%s)", hostName, cfg.Host, errc)
				continue
			}
			ctxDocker, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, errPing := d.Ping(ctxDocker); errPing != nil {
				log.Error(ctx, "hatchery> swarm> unable to ping docker host:%s", errPing)
				continue
			}
			log.Info(ctx, "hatchery> swarm> connected to %s (%s)", hostName, cfg.Host)

			h.dockerClients[hostName] = &dockerClient{
				Client:        *d,
				MaxContainers: cfg.MaxContainers,
				name:          hostName,
			}
		}
		if len(h.dockerClients) == 0 {
			log.Error(ctx, "hatchery> swarm> no docker host available. Please check errors")
			return fmt.Errorf("no docker engine available")
		}
	}

	if err := h.RefreshServiceLogger(ctx); err != nil {
		log.Error(ctx, "Hatchery> swarm> Cannot get cdn configuration : %v", err)
	}
	h.GoRoutines.Run(context.Background(), "swarm", func(ctx context.Context) {
		h.routines(ctx)
	})

	return nil
}

// SpawnWorker start a new docker container
// User can add option on prerequisite, as --port and --privileged
// but only hatchery NOT 'shared.infra' can launch containers with options
func (h *HatcherySwarm) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
	ctx, end := telemetry.Span(ctx, "swarm.SpawnWorker")
	defer end()

	if spawnArgs.JobID == 0 && !spawnArgs.RegisterOnly {
		return sdk.WithStack(fmt.Errorf("unable to spawn worker, no Job ID and no Register."))
	}

	telemetry.Current(ctx, telemetry.Tag(telemetry.TagWorker, spawnArgs.WorkerName))
	log.Debug("hatchery> swarm> SpawnWorker> Spawning worker %s", spawnArgs.WorkerName)

	// Choose a dockerEngine
	var dockerClient *dockerClient
	var foundDockerClient bool

	//  To choose a docker client by the number of containers
	fillrate := float64(-1)

	_, next := telemetry.Span(ctx, "swarm.chooseDockerEngine")
	for dname, dclient := range h.dockerClients {
		ctxList, cancelList := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelList()

		containers, errc := dclient.ContainerList(ctxList, types.ContainerListOptions{All: true})
		if errc != nil {
			log.Error(ctx, "hatchery> swarm> SpawnWorker> unable to list containers on %s: %v", dname, errc)
			continue
		}

		if len(containers) == 0 {
			dockerClient = h.dockerClients[dname]
			foundDockerClient = true
			break
		}

		var nbContainersFromHatchery int
		for _, cont := range containers {
			if _, ok := cont.Labels["hatchery"]; ok {
				nbContainersFromHatchery++
			}
		}

		// If client has enough space to start a container
		if nbContainersFromHatchery < h.dockerClients[dname].MaxContainers {
			clientFillRate := float64(nbContainersFromHatchery) / float64(h.dockerClients[dname].MaxContainers)
			if fillrate > clientFillRate || fillrate == -1 {
				fillrate = clientFillRate
				dockerClient = h.dockerClients[dname]
				foundDockerClient = true
			}
			if fillrate == 0 {
				break
			}
		}
	}
	next()

	if !foundDockerClient {
		return fmt.Errorf("unable to found suitable docker engine")
	}

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
					log.Warning(ctx, "hatchery> swarm> SpawnWorker>Unable to parse memory requirement %d :%v", memory, err)
					return err
				}
			} else if r.Type == sdk.ServiceRequirement {
				//Create a network if not already created
				if network == "" {
					network = spawnArgs.WorkerName + "-net"
					networkAlias = "worker"
					if err := h.createNetwork(ctx, dockerClient, network); err != nil {
						log.Warning(ctx, "hatchery> swarm> SpawnWorker> Unable to create network %s on %s for jobID %d : %v", network, dockerClient.name, spawnArgs.JobID, err)
						next()
						return err
					}
				}
				//name= <alias> => the name of the host put in /etc/hosts of the worker
				//value= "postgres:latest env_1=blabla env_2=blabla" => we can add env variables in requirement name
				img, envm := hatchery.ParseRequirementModel(r.Value)

				serviceMemory := int64(1024)
				if sm, ok := envm["CDS_SERVICE_MEMORY"]; ok {
					i, err := strconv.ParseUint(sm, 10, 32)
					if err != nil {
						log.Warning(ctx, "SpawnWorker> Unable to parse service option CDS_SERVICE_MEMORY=%s : %s", sm, err)
					} else {
						// too low values are checked in HatcherySwarm.createAndStartContainer() below
						serviceMemory = int64(i)
					}
				}

				var cmdArgs []string
				if sa, ok := envm["CDS_SERVICE_ARGS"]; ok {
					cmdArgs = hatchery.ParseArgs(sa)
				}
				if cmdArgs == nil {
					cmdArgs = []string{}
				}

				env := make([]string, 0, len(envm))
				for key, val := range envm {
					env = append(env, key+"="+val)
				}

				serviceName := r.Name + "-" + spawnArgs.WorkerName

				//labels are used to make container cleanup easier. We "link" the service to its worker this way.
				labels := map[string]string{
					"service_worker": spawnArgs.WorkerName,
					"service_name":   serviceName,
					"hatchery":       h.Config.Name,
				}

				if spawnArgs.JobID > 0 {
					labels[hatchery.LabelServiceProjectKey] = spawnArgs.ProjectKey
					labels[hatchery.LabelServiceWorkflowName] = spawnArgs.WorkflowName
					labels[hatchery.LabelServiceWorkflowID] = fmt.Sprintf("%d", spawnArgs.WorkflowID)
					labels[hatchery.LabelServiceRunID] = fmt.Sprintf("%d", spawnArgs.RunID)
					labels[hatchery.LabelServiceNodeRunID] = fmt.Sprintf("%d", spawnArgs.NodeRunID)
					labels[hatchery.LabelServiceNodeRunName] = spawnArgs.NodeRunName
					labels[hatchery.LabelServiceJobName] = spawnArgs.JobName
					labels[hatchery.LabelServiceJobID] = fmt.Sprintf("%d", spawnArgs.JobID)
					labels[hatchery.LabelServiceID] = fmt.Sprintf("%d", r.ID)
					labels[hatchery.LabelServiceReqName] = r.Name

				}

				//Start the services
				args := containerArgs{
					name:         serviceName,
					image:        img,
					network:      network,
					networkAlias: r.Name,
					cmd:          cmdArgs,
					env:          env,
					labels:       labels,
					memory:       serviceMemory,
					entryPoint:   nil,
				}

				if err := h.createAndStartContainer(ctx, dockerClient, args, spawnArgs); err != nil {
					log.Warning(ctx, "hatchery> swarm> SpawnWorker> Unable to start required container on %s: %s", dockerClient.name, err)
					return err
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
		"worker_model_path":   spawnArgs.Model.Group.Name + "/" + spawnArgs.Model.Name,
		"worker_name":         spawnArgs.WorkerName,
		"worker_requirements": strings.Join(services, ","),
		"hatchery":            h.Config.Name,
	}

	// Add new options on hatchery swarm to allow advanced docker option such as addHost, priviledge, port mapping and so one: #4594
	dockerOpts, errDockerOpts := h.computeDockerOpts(spawnArgs.Requirements)
	if errDockerOpts != nil {
		return errDockerOpts
	}

	udataParam := sdk.WorkerArgs{
		API:               h.Config.API.HTTP.URL,
		Token:             spawnArgs.WorkerToken,
		HTTPInsecure:      h.Config.API.HTTP.Insecure,
		Name:              spawnArgs.WorkerName,
		Model:             spawnArgs.Model.Group.Name + "/" + spawnArgs.Model.Name,
		TTL:               h.Config.WorkerTTL,
		HatcheryName:      h.Name(),
		GraylogHost:       h.Config.Provision.WorkerLogsOptions.Graylog.Host,
		GraylogPort:       h.Config.Provision.WorkerLogsOptions.Graylog.Port,
		GraylogExtraKey:   h.Config.Provision.WorkerLogsOptions.Graylog.ExtraKey,
		GraylogExtraValue: h.Config.Provision.WorkerLogsOptions.Graylog.ExtraValue,
	}

	udataParam.WorkflowJobID = spawnArgs.JobID

	tmpl, errt := template.New("cmd").Parse(spawnArgs.Model.ModelDocker.Cmd)
	if errt != nil {
		return errt
	}
	var buffer bytes.Buffer
	if errTmpl := tmpl.Execute(&buffer, udataParam); errTmpl != nil {
		return errTmpl
	}
	cmds := strings.Fields(spawnArgs.Model.ModelDocker.Shell)
	cmds = append(cmds, buffer.String())

	// copy envs to avoid data race
	modelEnvs := make(map[string]string, len(spawnArgs.Model.ModelDocker.Envs))
	for k, v := range spawnArgs.Model.ModelDocker.Envs {
		modelEnvs[k] = v
	}

	envsWm := map[string]string{}
	envsWm["CDS_MODEL_MEMORY"] = fmt.Sprintf("%d", memory)
	envsWm["CDS_API"] = udataParam.API
	envsWm["CDS_TOKEN"] = udataParam.Token
	envsWm["CDS_NAME"] = udataParam.Name
	envsWm["CDS_MODEL_PATH"] = udataParam.Model
	envsWm["CDS_HATCHERY_NAME"] = udataParam.HatcheryName
	envsWm["CDS_FROM_WORKER_IMAGE"] = fmt.Sprintf("%v", udataParam.FromWorkerImage)
	envsWm["CDS_INSECURE"] = fmt.Sprintf("%v", udataParam.HTTPInsecure)

	if spawnArgs.JobID > 0 {
		envsWm["CDS_BOOKED_WORKFLOW_JOB_ID"] = fmt.Sprintf("%d", spawnArgs.JobID)
	}

	envTemplated, errEnv := sdk.TemplateEnvs(udataParam, modelEnvs)
	if errEnv != nil {
		return errEnv
	}

	for envName, envValue := range envTemplated {
		envsWm[envName] = envValue
	}

	envs := make([]string, len(envsWm))
	i := 0
	for envName, envValue := range envsWm {
		envs[i] = envName + "=" + envValue
		i++
	}

	args := containerArgs{
		name:         spawnArgs.WorkerName,
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
	if err := h.createAndStartContainer(ctx, dockerClient, args, spawnArgs); err != nil {
		log.Warning(ctx, "hatchery> swarm> SpawnWorker> Unable to start container %s on %s with image %s err:%v", args.name, dockerClient.name, spawnArgs.Model.ModelDocker.Image, err)
		return err
	}

	return nil
}

// ModelType returns type of hatchery
func (*HatcherySwarm) ModelType() string {
	return sdk.Docker
}

const (
	timeoutPullImage = 10 * time.Minute
)

// CanSpawn checks if the model can be spawned by this hatchery
// it checks on every docker engine is one of the docker has availability
func (h *HatcherySwarm) CanSpawn(ctx context.Context, model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	// Hostname requirement are not supported
	for _, r := range requirements {
		if r.Type == sdk.HostnameRequirement {
			log.Debug("CanSpawn> Job %d has a hostname requirement. Swarm can't spawn a worker for this job", jobID)
			return false
		}
	}
	for dockerName, dockerClient := range h.dockerClients {
		//List all containers to check if we can spawn a new one
		cs, errList := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if errList != nil {
			log.Error(ctx, "hatchery> swarm> CanSpawn> Unable to list containers on %s: %s", dockerName, errList)
			continue
		}

		var nbContainersFromHatchery int
		for _, cont := range cs {
			if _, ok := cont.Labels["hatchery"]; ok {
				nbContainersFromHatchery++
			}
		}

		//List all workers
		ws, errWList := h.getWorkerContainers(cs, types.ContainerListOptions{})
		if errWList != nil {
			log.Error(ctx, "hatchery> swarm> CanSpawn> Unable to list workers on %s: %s", dockerName, errWList)
			continue
		}

		//Checking the number of container on each docker engine
		if nbContainersFromHatchery >= dockerClient.MaxContainers {
			log.Debug("hatchery> swarm> CanSpawn> max containers reached on %s. current:%d max:%d", dockerName, nbContainersFromHatchery, dockerClient.MaxContainers)
			continue
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
			ratioService := h.Config.Provision.RatioService
			if ratioService != nil && *ratioService >= 100 {
				log.Debug("hatchery> swarm> CanSpawn> ratioService 100 by conf on %s - no spawn worker without CDS Service", dockerName)
				return false
			}
			if nbContainersFromHatchery > 0 {
				percentFree := 100 - (100 * len(ws) / dockerClient.MaxContainers)
				if ratioService != nil && percentFree <= *ratioService {
					log.Debug("hatchery> swarm> CanSpawn> ratio reached on %s. percentFree:%d ratioService:%d", dockerName, percentFree, *ratioService)
					return false
				}
			}
		}
		return true
	}
	return false
}

func (h *HatcherySwarm) getWorkerContainers(containers []types.Container, option types.ContainerListOptions) ([]types.Container, error) {
	res := []types.Container{}
	//We only count worker
	for _, cont := range containers {
		if _, ok := cont.Labels["worker_name"]; ok {
			if hatch, ok := cont.Labels["hatchery"]; !ok || hatch == h.Config.Name {
				res = append(res, cont)
			}
		}
	}
	return res, nil
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcherySwarm) WorkersStarted(ctx context.Context) []string {
	res := make([]string, 0)
	for _, dockerClient := range h.dockerClients {
		// get only started containers
		containers, errList := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if errList != nil {
			log.Error(ctx, "hatchery> swarm> WorkersStarted> Unable to list containers: %s", errList)
			return nil
		}
		workers, _ := h.getWorkerContainers(containers, types.ContainerListOptions{})
		for _, w := range workers {
			res = append(res, w.Labels["worker_name"])
		}
	}
	return res
}

// WorkersStartedByModel returns the number of started workers
func (h *HatcherySwarm) WorkersStartedByModel(ctx context.Context, model *sdk.Model) int {
	list := []string{}
	for _, dockerClient := range h.dockerClients {
		// get only started containers
		containers, errList := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if errList != nil {
			log.Error(ctx, "hatchery> swarm> WorkersStartedByModel> Unable to list containers: %s", errList)
			return 0
		}
		workers, errList := h.getWorkerContainers(containers, types.ContainerListOptions{})
		if errList != nil {
			log.Error(ctx, "hatchery> swarm> WorkersStartedByModel> Unable to list containers: %s", errList)
			return 0
		}

		for _, c := range workers {
			log.Debug("Container : %s %s [%s]", c.ID, c.Image, c.Status)
			if c.Image == model.ModelDocker.Image {
				list = append(list, c.ID)
			}
		}
	}
	log.Debug("hatchery> swarm> WorkersStartedByModel> %s \t %d", model.Name, len(list))
	return len(list)
}

func (h *HatcherySwarm) GetLogger() *logrus.Logger {
	return h.ServiceLogger
}

// Serve start the hatchery server
func (h *HatcherySwarm) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcherySwarm) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// WorkerModelsEnabled returns Worker model enabled
func (h *HatcherySwarm) WorkerModelsEnabled() ([]sdk.Model, error) {
	return h.CDSClient().WorkerModelEnabledList()
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcherySwarm) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

func (h *HatcherySwarm) routines(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.GoRoutines.Exec(ctx, "getServicesLogs", func(ctx context.Context) {
				if err := h.getServicesLogs(); err != nil {
					log.Error(ctx, "Hatchery> swarm> Cannot get service logs : %v", err)
				}
			})

			h.GoRoutines.Exec(ctx, "killAwolWorker", func(ctx context.Context) {
				_ = h.killAwolWorker(ctx)
			})

			h.GoRoutines.Exec(ctx, "refreshCDNConfiguration", func(ctx context.Context) {
				if err := h.RefreshServiceLogger(ctx); err != nil {
					log.Error(ctx, "Hatchery> swarm> Cannot get cdn configuration : %v", err)
				}
			})
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Hatchery> Swarm> Exiting routines")
			}
			return
		}
	}
}

func (h *HatcherySwarm) listAwolWorkers(dockerClientName string, containers []types.Container) ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apiworkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot get workers on %s", dockerClientName)
	}

	workers, errList := h.getWorkerContainers(containers, types.ContainerListOptions{All: true})
	if errList != nil {
		return nil, sdk.WrapError(err, "Cannot list containers on %s", dockerClientName)
	}

	//Checking workers
	oldContainers := []types.Container{}
	for _, c := range workers {
		if !strings.Contains(c.Status, "Exited") && time.Now().Add(-3*time.Minute).Unix() < c.Created {
			log.Debug("hatchery> swarm> listAwolWorkers> container %s(status=%s) is too young", c.Names[0], c.Status)
			continue
		}

		//If there isn't any worker registered on the API. Kill the container
		if len(apiworkers) == 0 {
			log.Debug("hatchery> swarm> listAwolWorkers> no apiworkers returned by api container %s will be deleted", c.Names[0])
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
					log.Debug("hatchery> swarm> listAwolWorkers> Worker %s is disabled. Kill it with fire!", c.Names[0])
					oldContainers = append(oldContainers, c)
					break
				}
			}
		}
		//If the container doesn't match any worker : Kill it.
		if !found {
			log.Debug("hatchery> swarm> listAwolWorkers> container %s not found on apiworkers", c.Names[0])
			oldContainers = append(oldContainers, c)
		}
	}

	return oldContainers, nil
}

func (h *HatcherySwarm) killAwolWorker(ctx context.Context) error {
	for _, dockerClient := range h.dockerClients {
		containers, errC := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if errC != nil {
			log.Warning(ctx, "hatchery> swarm> killAwolWorker> Cannot list containers: %s on %s", errC, dockerClient.name)
			return errC
		}

		oldContainers, err := h.listAwolWorkers(dockerClient.name, containers)
		if err != nil {
			log.Warning(ctx, "hatchery> swarm> killAwolWorker> Cannot list workers %s on %s", err, dockerClient.name)
			return err
		}

		// Delete the workers
		for _, c := range oldContainers {
			log.Debug("hatchery> swarm> killAwolWorker> Delete worker %s on %s", c.Names[0], dockerClient.name)
			if err := h.killAndRemove(ctx, dockerClient, c.ID, containers); err != nil {
				log.Debug("hatchery> swarm> killAwolWorker> %v", err)
			}
		}

		// creating a map of containers names
		mContainers := map[string]struct{}{}
		for i := range containers {
			name := strings.TrimPrefix(containers[i].Names[0], "/") // docker returns name prefixed by a /
			mContainers[name] = struct{}{}
		}

		// Checking services
		for _, c := range containers {
			// checks if the container is a service based on its labels
			if c.Labels["service_worker"] == "" {
				continue
			}
			// if the worker associated to this service is still alive do not kill the service
			if _, workerStillAlive := mContainers[c.Labels["service_worker"]]; workerStillAlive {
				continue
			}

			if !strings.Contains(c.Status, "Exited") && time.Now().Add(-3*time.Minute).Unix() < c.Created {
				log.Debug("hatchery> swarm> killAwolWorker> container %s(status=%s) is too young - service associated to worker %s", c.Names[0], c.Status, c.Labels["service_worker"])
				continue
			}

			// Send final logs before deleting service container
			jobIdentifiers := h.GetIdentifiersFromLabels(c)
			if jobIdentifiers == nil {
				continue
			}
			endLog := log.Message{
				Level: logrus.InfoLevel,
				Value: string("End of Job"),
				Signature: log.Signature{
					Service: &log.SignatureService{
						HatcheryID:      h.Service().ID,
						HatcheryName:    h.ServiceName(),
						RequirementID:   jobIdentifiers.ServiceID,
						RequirementName: c.Labels[hatchery.LabelServiceReqName],
						WorkerName:      c.Labels["service_worker"],
					},
					ProjectKey:   c.Labels[hatchery.LabelServiceProjectKey],
					WorkflowName: c.Labels[hatchery.LabelServiceWorkflowName],
					WorkflowID:   jobIdentifiers.WorkflowID,
					RunID:        jobIdentifiers.RunID,
					NodeRunName:  c.Labels[hatchery.LabelServiceNodeRunName],
					JobName:      c.Labels[hatchery.LabelServiceJobName],
					JobID:        jobIdentifiers.JobID,
					NodeRunID:    jobIdentifiers.NodeRunID,
					Timestamp:    time.Now().UnixNano(),
				},
			}
			h.Common.SendServiceLog(ctx, []log.Message{endLog}, sdk.StatusTerminated)

			log.Debug("hatchery> swarm> killAwolWorker> Delete worker (service) %s on %s", c.Names[0], dockerClient.name)
			if err := h.killAndRemove(ctx, dockerClient, c.ID, containers); err != nil {
				log.Error(ctx, "hatchery> swarm> killAwolWorker> service %v on %s", err, dockerClient.name)
			}
			continue
		}
	}
	return h.killAwolNetworks(ctx)
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcherySwarm) NeedRegistration(ctx context.Context, m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
