package openstack

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/slug"
)

var (
	workersAlive map[string]int64

	ipsInfos = struct {
		mu  sync.RWMutex
		ips map[string]ipInfos
	}{
		mu:  sync.RWMutex{},
		ips: map[string]ipInfos{},
	}
)

var _ hatchery.InterfaceWithModels = new(HatcheryOpenstack)

// New instanciates a new Hatchery Openstack
func New() *HatcheryOpenstack {
	s := new(HatcheryOpenstack)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

// Init cdsclient config.
func (h *HatcheryOpenstack) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid openstack hatchery configuration"))
	}
	h.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: sConfig.HTTP,
	}
	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.TokenV2 = sConfig.API.TokenV2
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryOpenstack) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	h.Common.Common.ServiceName = h.Config.Name
	h.Common.Common.ServiceType = sdk.TypeHatchery
	h.HTTPURL = h.Config.URL

	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	var err error
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}
	h.Common.Common.Region = h.Config.Provision.Region
	h.Common.Common.IgnoreJobWithNoRegion = h.Config.Provision.IgnoreJobWithNoRegion
	h.Common.Common.ModelType = h.ModelType()

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcheryOpenstack) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	ws, err := h.WorkersStarted(ctx)
	if err != nil {
		ctx = log.ContextWithStackTrace(ctx, err)
		log.Warn(ctx, err.Error())
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(ws), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryOpenstack) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid hatchery openstack configuration")
	}

	if err := hconfig.Check(); err != nil {
		return fmt.Errorf("Invalid hatchery openstack configuration: %v", err)
	}

	if hconfig.Tenant == "" && hconfig.Domain == "" {
		return fmt.Errorf("One of Openstack-tenant (auth v2) or Openstack-domain (auth v3) is mandatory")
	}

	if hconfig.User == "" {
		return fmt.Errorf("Openstack-user is mandatory")
	}

	if hconfig.Address == "" {
		return fmt.Errorf("Openstack-auth-endpoint is mandatory")
	}

	if hconfig.Password == "" {
		return fmt.Errorf("Openstack-password is mandatory")
	}

	if hconfig.Region == "" {
		return fmt.Errorf("Openstack-region is mandatory")
	}

	if hconfig.IPRange != "" {
		ips, err := sdk.IPinRanges(context.Background(), hconfig.IPRange)
		if err != nil {
			return fmt.Errorf("flag or environment variable openstack-ip-range error: %s", err)
		}
		for _, ip := range ips {
			ipsInfos.ips[ip] = ipInfos{}
		}
	}

	return nil
}

func (h *HatcheryOpenstack) Signin(ctx context.Context, clientConfig cdsclient.ServiceConfig, srvConfig interface{}) error {
	if err := h.Common.Signin(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	if err := h.Common.SigninV2(ctx, clientConfig, srvConfig); err != nil {
		return err
	}
	return nil
}

// Start inits client and routines for hatchery
func (h *HatcheryOpenstack) Start(ctx context.Context) error {
	return hatchery.Create(ctx, h)
}

// Serve start the hatchery server
func (h *HatcheryOpenstack) Serve(ctx context.Context) error {
	return h.CommonServe(ctx, h)
}

// Configuration returns Hatchery CommonConfiguration
func (h *HatcheryOpenstack) Configuration() service.HatcheryCommonConfiguration {
	return h.Config.HatcheryCommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryOpenstack) ModelType() string {
	return sdk.Openstack
}

// WorkerModelsEnabled returns Worker model enabled.
func (h *HatcheryOpenstack) WorkerModelsEnabled() ([]sdk.Model, error) {
	allModels, err := h.CDSClient().WorkerModelEnabledList()
	if err != nil {
		return nil, err
	}

	filteredModels := make([]sdk.Model, 0, len(allModels))
	for i := range allModels {
		if allModels[i].Type != sdk.Openstack {
			continue
		}

		// Required flavor should be available on target OpenStack project
		if _, err := h.flavor(allModels[i].ModelVirtualMachine.Flavor); err != nil {
			log.Debug(context.TODO(), "WorkerModelsEnabled> model %s/%s is not usable because flavor '%s' not found", allModels[i].Group.Name, allModels[i].Name, allModels[i].ModelVirtualMachine.Flavor)
			continue
		}

		filteredModels = append(filteredModels, allModels[i])
	}

	// Sort models by required CPUs, this will allows to starts job without defined model on the smallest flavor.
	sort.Slice(filteredModels, func(i, j int) bool {
		flavorI, _ := h.flavor(filteredModels[i].ModelVirtualMachine.Flavor)
		flavorJ, _ := h.flavor(filteredModels[j].ModelVirtualMachine.Flavor)

		// If same flavor sort by model name
		if flavorI.Name == flavorJ.Name {
			return filteredModels[i].Name < filteredModels[j].Name
		}

		// Models with the default flavor should be at the beginning of the list
		if flavorI.Name == h.Config.DefaultFlavor {
			return true
		}
		if flavorJ.Name == h.Config.DefaultFlavor {
			return false
		}

		return flavorI.VCPUs < flavorJ.VCPUs
	})

	return filteredModels, nil
}

// WorkerModelSecretList returns secret for given model.
func (h *HatcheryOpenstack) WorkerModelSecretList(m sdk.Model) (sdk.WorkerModelSecrets, error) {
	return h.CDSClient().WorkerModelSecretList(m.Group.Name, m.Name)
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryOpenstack) CanSpawn(ctx context.Context, model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement || r.Type == sdk.HostnameRequirement {
			return false
		}
	}
	return true
}

func (h *HatcheryOpenstack) main(ctx context.Context) {
	serverListTick := time.NewTicker(10 * time.Second).C
	killAwolServersTick := time.NewTicker(30 * time.Second).C
	killErrorServersTick := time.NewTicker(60 * time.Second).C
	killDisabledWorkersTick := time.NewTicker(60 * time.Second).C

	for {
		select {
		case <-serverListTick:
			h.updateServerList(ctx)
		case <-killAwolServersTick:
			h.killAwolServers(ctx)
		case <-killErrorServersTick:
			h.killErrorServers(ctx)
		case <-killDisabledWorkersTick:
			h.killDisabledWorkers()
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Hatchery> openstack> Exiting routines")
			}
			return

		}
	}
}

func (h *HatcheryOpenstack) updateServerList(ctx context.Context) {
	var out string
	var total int
	status := map[string]int{}

	for _, s := range h.getServers(ctx) {
		out += fmt.Sprintf("- [%s] %s:%s ", s.Updated, s.Status, s.Name)
		if _, ok := status[s.Status]; !ok {
			status[s.Status] = 0
		}
		status[s.Status]++
		total++
	}
	var st string
	for k, s := range status {
		st += fmt.Sprintf("%d %s ", s, k)
	}
	log.Debug(ctx, "Got %d servers %s", total, st)
	if total > 0 {
		log.Debug(ctx, out)
	}
}

func (h *HatcheryOpenstack) killAwolServers(ctx context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := h.CDSClient().WorkerList(ctx)
	now := time.Now().Unix()
	if err != nil {
		log.Warn(ctx, "killAwolServers> Cannot fetch worker list: %s", err)
		return
	}

	for _, s := range h.getServers(ctx) {
		log.Debug(ctx, "killAwolServers> Checking %s %v", s.Name, s.Metadata)
		workerName, isWorker := s.Metadata["worker"]
		// if the vm is in BUILD state since > 15 min, we delete it
		if s.Status == "BUILD" {
			if isWorker && time.Since(s.Created) > 15*time.Minute {
				log.Warn(ctx, "killAwolServers> Deleting server %s status: %s last update: %s", s.Name, s.Status, time.Since(s.Updated))
				if err := h.deleteServer(ctx, s); err != nil {
					log.Error(ctx, "killAwolServers> Error while deleting server %s not created status: %s last update: %s", s.Name, s.Status, time.Since(s.Updated))
				}
			}
			continue
		}

		var inWorkersList bool
		for _, w := range workers {
			if _, ok := workersAlive[w.Name]; !ok {
				log.Debug(ctx, "killAwolServers> add %s to map workersAlive", w.Name)
				workersAlive[w.Name] = now
			}

			if w.Name == s.Name {
				inWorkersList = true
				workersAlive[w.Name] = now
				break
			}
		}

		workerHatcheryName, _ := s.Metadata["hatchery_name"]
		registerOnly, _ := s.Metadata["register_only"]
		workerModelPath, _ := s.Metadata["worker_model_path"]
		workerModelNameLastModified, _ := s.Metadata["worker_model_last_modified"]
		model, _ := s.Metadata["model"]
		flavor, _ := s.Metadata["flavor"]

		var toDeleteKilled bool
		if isWorker {
			if _, wasAlive := workersAlive[workerName]; wasAlive {
				if !inWorkersList {
					toDeleteKilled = true
					log.Debug(ctx, "killAwolServers> %s toDeleteKilled --> true", workerName)
					delete(workersAlive, workerName)
				}
			}
		}

		// Delete workers, if not identified by CDS API
		// Wait for 10 minutes, to avoid killing worker babies
		log.Debug(ctx, "killAwolServers> server %s status: %s last update: %s toDeleteKilled:%t inWorkersList:%t", s.Name, s.Status, time.Since(s.Updated), toDeleteKilled, inWorkersList)
		if isWorker && (workerHatcheryName == "" || workerHatcheryName == h.Name()) &&
			(s.Status == "SHUTOFF" || toDeleteKilled || (!inWorkersList && time.Since(s.Updated) > 10*time.Minute)) {

			// if it's was a worker model for registration
			// check if we need to create a new openstack image from it
			// by comparing userDateLastModified from worker model
			if !h.Config.DisableCreateImage && s.Status == "SHUTOFF" && registerOnly == "true" {
				h.killAwolServersComputeImage(ctx, workerModelPath, workerModelNameLastModified, s.ID, model, flavor)
			}

			log.Debug(ctx, "killAwolServers> Deleting server %s status: %s last update: %s registerOnly:%s toDeleteKilled:%t inWorkersList:%t", s.Name, s.Status, time.Since(s.Updated), registerOnly, toDeleteKilled, inWorkersList)
			_ = h.deleteServer(ctx, s)
		}
	}
	// then clean workersAlive map
	toDelete := []string{}
	for workerName, t := range workersAlive {
		if t != now {
			toDelete = append(toDelete, workerName)
		}
	}
	for _, workerName := range toDelete {
		delete(workersAlive, workerName)
	}
	log.Debug(ctx, "killAwolServers> workersAlive: %+v", workersAlive)
}

func (h *HatcheryOpenstack) killAwolServersComputeImage(ctx context.Context, workerModelPath, workerModelNameLastModified, serverID, model, flavor string) {
	oldImagesID := []string{}
	for _, img := range h.getImages(ctx) {
		if w := img.Metadata["worker_model_path"]; w == workerModelPath {
			oldImagesID = append(oldImagesID, img.ID)
			if d, ok := img.Metadata["worker_model_last_modified"]; ok && d.(string) == workerModelNameLastModified {
				// no need to recreate an image
				return
			}
		}
	}

	log.Info(ctx, "killAwolServersComputeImage> create image before deleting server")
	imageID, err := servers.CreateImage(h.openstackClient, serverID, servers.CreateImageOpts{
		Name: "cds_image_" + slug.Convert(workerModelPath),
		Metadata: map[string]string{
			"worker_model_path":          workerModelPath,
			"model":                      model,
			"flavor":                     flavor,
			"created_by":                 "cdsHatchery_" + h.Name(),
			"worker_model_last_modified": workerModelNameLastModified,
		},
	}).ExtractImageID()
	if err != nil {
		log.Error(ctx, "killAwolServersComputeImage> error on create image for worker model %s: %s", workerModelPath, err)
	} else {
		log.Info(ctx, "killAwolServersComputeImage> image %s created for worker model %s - waiting %ds for saving created img...", imageID, workerModelPath, h.Config.CreateImageTimeout)

		startTime := time.Now().Unix()
		var newImageIsActive bool
		for time.Now().Unix()-startTime < int64(h.Config.CreateImageTimeout) {
			newImage, err := images.Get(h.openstackClient, imageID).Extract()
			if err != nil {
				log.Error(ctx, "killAwolServersComputeImage> error on get new image %s for worker model %s: %s", imageID, workerModelPath, err)
			}
			if newImage.Status == "ACTIVE" {
				// new image is created, end wait
				log.Info(ctx, "killAwolServersComputeImage> image %s created for worker model %s is active", imageID, workerModelPath)
				newImageIsActive = true
				break
			}
			time.Sleep(15 * time.Second)
		}

		if !newImageIsActive {
			log.Info(ctx, "killAwolServersComputeImage> timeout while creating new image. Deleting new image for %s with ID %s", workerModelPath, imageID)
			if err := images.Delete(h.openstackClient, imageID).ExtractErr(); err != nil {
				log.Error(ctx, "killAwolServersComputeImage> error while deleting new image %s", imageID)
			}
		}

		for _, oldImageID := range oldImagesID {
			log.Info(ctx, "killAwolServersComputeImage> deleting old image for %s with ID %s", workerModelPath, oldImageID)
			if err := images.Delete(h.openstackClient, oldImageID).ExtractErr(); err != nil {
				log.Error(ctx, "killAwolServersComputeImage> error while deleting old image %s", oldImageID)
			}
		}

		h.resetImagesCache()
	}
}

func (h *HatcheryOpenstack) killErrorServers(ctx context.Context) {
	for _, s := range h.getServers(ctx) {
		//Remove server without IP Address
		if s.Status == "ACTIVE" {
			if len(s.Addresses) == 0 && time.Since(s.Updated) > 10*time.Minute {
				log.Info(ctx, "killErrorServers> len(s.Addresses):%d s.Updated: %v", len(s.Addresses), time.Since(s.Updated))
				_ = h.deleteServer(ctx, s)
			}
		}

		//Remove Error server
		if s.Status == "ERROR" {
			log.Info(ctx, "killErrorServers> s.Status: %s", s.Status)
			_ = h.deleteServer(ctx, s)
		}
	}
}

func (h *HatcheryOpenstack) killDisabledWorkers() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workerPoolDisabled, err := hatchery.WorkerPool(ctx, h, sdk.StatusDisabled)
	if err != nil {
		log.Error(ctx, "killDisabledWorkers> Pool> Error: %v", err)
		return
	}

	srvs := h.getServers(ctx)

	for _, w := range workerPoolDisabled {
		for _, s := range srvs {
			if s.Name == w.Name {
				log.Info(ctx, "killDisabledWorkers> killDisabledWorkers %v", s.Name)
				_ = h.deleteServer(ctx, s)
				break
			}
		}
	}
}

func (h *HatcheryOpenstack) deleteServer(ctx context.Context, s servers.Server) error {
	log.Info(ctx, "Deleting worker %s", s.Name)

	// If its a worker "register", check registration before deleting it
	if strings.HasPrefix(s.Name, "register-") {
		modelPath := s.Metadata["worker_model_path"]
		//Send registering logs....
		consoleLog, err := h.getConsoleLog(s)
		if err != nil {
			log.Error(ctx, "killAndRemove> unable to get console log from registering server %s: %v", s.Name, err)
		}
		if err := hatchery.CheckWorkerModelRegister(ctx, h, modelPath); err != nil {
			var spawnErr = sdk.SpawnErrorForm{
				Error: err.Error(),
				Logs:  []byte(consoleLog),
			}
			tuple := strings.SplitN(modelPath, "/", 2)
			if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
				log.Error(ctx, "CheckWorkerModelRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", modelPath, spawnErr)
			}
		}
	}

	r := servers.Delete(h.openstackClient, s.ID)
	if err := r.ExtractErr(); err != nil {
		log.Warn(ctx, "deleteServer> Cannot delete worker %s: %s", s.Name, err)
		return err
	}
	return nil
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryOpenstack) WorkersStarted(ctx context.Context) ([]string, error) {
	srvs := h.getServers(ctx)
	res := make([]string, len(srvs))
	for i, s := range srvs {
		res[i] = s.Metadata["worker"]
	}
	return res, nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryOpenstack) NeedRegistration(ctx context.Context, m *sdk.Model) bool {
	if m.NeedRegistration {
		log.Debug(ctx, "NeedRegistration> true as worker model %s model.NeedRegistration=true", m.Name)
		return true
	}
	for _, img := range h.getImages(ctx) {
		if w := img.Metadata["worker_model_path"]; w == m.Group.Name+"/"+m.Name {
			if d, ok := img.Metadata["worker_model_last_modified"]; ok {
				if fmt.Sprintf("%d", m.UserLastModified.Unix()) == d.(string) {
					log.Debug(ctx, "NeedRegistration> false. An image is already available for this worker model %s workerModel.UserLastModified", m.Name)
					return false
				}
			}
		}
	}
	log.Debug(ctx, "NeedRegistration> true. No existing image found for this worker model %s", m.Name)
	return true
}
