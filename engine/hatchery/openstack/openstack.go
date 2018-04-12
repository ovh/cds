package openstack

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

var (
	hatcheryOpenStack *HatcheryOpenstack

	workersAlive map[string]int64

	ipsInfos = struct {
		mu  sync.RWMutex
		ips map[string]ipInfos
	}{
		mu:  sync.RWMutex{},
		ips: map[string]ipInfos{},
	}
)

// New instanciates a new Hatchery Openstack
func New() *HatcheryOpenstack {
	return new(HatcheryOpenstack)
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

	// h.Client = cdsclient.NewService(h.Config.API.HTTP.URL, 60*time.Second)
	// h.API = h.Config.API.HTTP.URL
	// h.Name = h.Config.Name
	// h.HTTPURL = h.Config.URL
	// h.Token = h.Config.API.Token
	// h.Type = services.TypeHatchery
	// h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryOpenstack) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if hconfig.API.HTTP.URL == "" {
		return fmt.Errorf("API HTTP(s) URL is mandatory")
	}

	if hconfig.API.Token == "" {
		return fmt.Errorf("API Token URL is mandatory")
	}

	if hconfig.Tenant == "" {
		return fmt.Errorf("Openstack-tenant is mandatory")
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

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your openstack hatchery configuration")
	}

	if hconfig.IPRange != "" {
		ips, err := IPinRanges(hconfig.IPRange)
		if err != nil {
			return fmt.Errorf("flag or environment variable openstack-ip-range error: %s", err)
		}
		for _, ip := range ips {
			ipsInfos.ips[ip] = ipInfos{}
		}
	}

	return nil
}

// Serve start the HatcheryOpenstack server
func (h *HatcheryOpenstack) Serve(ctx context.Context) error {
	return hatchery.Create(h)
}

// ID returns hatchery id
func (h *HatcheryOpenstack) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryOpenstack) Hatchery() *sdk.Hatchery {
	return h.hatch
}

//Client returns cdsclient instance
func (h *HatcheryOpenstack) Client() cdsclient.Interface {
	return h.client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryOpenstack) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryOpenstack) ModelType() string {
	return sdk.Openstack
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryOpenstack) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	return true
}

func (h *HatcheryOpenstack) main() {
	serverListTick := time.NewTicker(10 * time.Second).C
	killAwolServersTick := time.NewTicker(30 * time.Second).C
	killErrorServersTick := time.NewTicker(60 * time.Second).C
	killDisabledWorkersTick := time.NewTicker(60 * time.Second).C

	for {
		select {

		case <-serverListTick:
			h.updateServerList()

		case <-killAwolServersTick:
			h.killAwolServers()

		case <-killErrorServersTick:
			h.killErrorServers()

		case <-killDisabledWorkersTick:
			h.killDisabledWorkers()

		}
	}
}

func (h *HatcheryOpenstack) updateServerList() {
	var out string
	var total int
	status := map[string]int{}

	for _, s := range h.getServers() {
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
	log.Info("Got %d servers %s", total, st)
	if total > 0 {
		log.Debug(out)
	}
}

func (h *HatcheryOpenstack) killAwolServers() {
	workers, err := h.Client().WorkerList()
	now := time.Now().Unix()
	if err != nil {
		log.Warning("killAwolServers> Cannot fetch worker list: %s", err)
		return
	}

	for _, s := range h.getServers() {
		log.Debug("killAwolServers> Checking %s %v", s.Name, s.Metadata)
		if s.Status == "BUILD" {
			continue
		}

		var inWorkersList bool
		for _, w := range workers {
			if _, ok := workersAlive[w.Name]; !ok {
				log.Debug("killAwolServers> add %s to map workersAlive", w.Name)
				workersAlive[w.Name] = now
			}

			if w.Name == s.Name {
				inWorkersList = true
				workersAlive[w.Name] = now
				break
			}
		}

		workerHatcheryName, _ := s.Metadata["hatchery_name"]
		workerName, isWorker := s.Metadata["worker"]
		registerOnly, _ := s.Metadata["register_only"]
		workerModelName, _ := s.Metadata["worker_model_name"]
		workerModelNameLastModified, _ := s.Metadata["worker_model_last_modified"]
		model, _ := s.Metadata["model"]
		flavor, _ := s.Metadata["flavor"]

		var toDeleteKilled bool
		if isWorker {
			if _, wasAlive := workersAlive[workerName]; wasAlive {
				if !inWorkersList {
					toDeleteKilled = true
					log.Debug("killAwolServers> %s toDeleteKilled --> true", workerName)
					delete(workersAlive, workerName)
				}
			}
		}

		// Delete workers, if not identified by CDS API
		// Wait for 6 minutes, to avoid killing worker babies
		log.Debug("killAwolServers> server %s status: %s last update: %s toDeleteKilled:%t inWorkersList:%t", s.Name, s.Status, time.Since(s.Updated), toDeleteKilled, inWorkersList)
		if isWorker && (workerHatcheryName == "" || workerHatcheryName == h.Hatchery().Name) &&
			(s.Status == "SHUTOFF" || toDeleteKilled || (!inWorkersList && time.Since(s.Updated) > 6*time.Minute)) {

			// if it's was a worker model for registration
			// check if we need to create a new openstack image from it
			// by comparing userDateLastModified from worker model
			if !h.Config.DisableCreateImage && s.Status == "SHUTOFF" && registerOnly == "true" {
				h.killAwolServersComputeImage(workerModelName, workerModelNameLastModified, s.ID, model, flavor)
			}

			log.Debug("killAwolServers> Deleting server %s status: %s last update: %s registerOnly:%s toDeleteKilled:%t inWorkersList:%t", s.Name, s.Status, time.Since(s.Updated), registerOnly, toDeleteKilled, inWorkersList)
			if err := servers.Delete(h.openstackClient, s.ID).ExtractErr(); err != nil {
				log.Warning("killAwolServers> Cannot remove server %s: %s", s.Name, err)
				continue
			}
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
	log.Debug("killAwolServers> workersAlive: %+v", workersAlive)
}

func (h *HatcheryOpenstack) killAwolServersComputeImage(workerModelName, workerModelNameLastModified, serverID, model, flavor string) {
	var oldImageID string
	var oldDateLastModified string
	for _, img := range h.getImages() {
		w, _ := img.Metadata["worker_model_name"]
		if w == workerModelName {
			oldImageID = img.ID
			if d, ok := img.Metadata["worker_model_last_modified"]; ok {
				oldDateLastModified = d.(string)
			}
		}
	}

	if oldDateLastModified == workerModelNameLastModified {
		// no need to recreate an image
		return
	}

	log.Info("killAwolServersComputeImage> create image before deleting server")
	imageID, err := servers.CreateImage(h.openstackClient, serverID, servers.CreateImageOpts{
		Name: "cds_image_" + workerModelName,
		Metadata: map[string]string{
			"worker_model_name":          workerModelName,
			"model":                      model,
			"flavor":                     flavor,
			"created_by":                 "cdsHatchery_" + h.Hatchery().Name,
			"worker_model_last_modified": workerModelNameLastModified,
		},
	}).ExtractImageID()
	if err != nil {
		log.Error("killAwolServersComputeImage> error on create image for worker model %s: %s", workerModelName, err)
	} else {
		log.Info("killAwolServersComputeImage> image %s created for worker model %s - waiting %ds for saving created img...", imageID, workerModelName, h.Config.CreateImageTimeout)

		startTime := time.Now().Unix()
		var newImageIsActive bool
		for time.Now().Unix()-startTime < int64(h.Config.CreateImageTimeout) {
			newImage, err := images.Get(h.openstackClient, imageID).Extract()
			if err != nil {
				log.Error("killAwolServersComputeImage> error on get new image %s for worker model %s: %s", imageID, workerModelName, err)
			}
			if newImage.Status == "ACTIVE" {
				// new image is created, end wait
				log.Info("killAwolServersComputeImage> image %s created for worker model %s is active", imageID, workerModelName)
				newImageIsActive = true
				break
			}
			time.Sleep(15 * time.Second)
		}

		if !newImageIsActive {
			log.Info("killAwolServersComputeImage> timeout while creating new image. Deleting new image for %s with ID %s", workerModelName, imageID)
			if err := images.Delete(h.openstackClient, imageID).ExtractErr(); err != nil {
				log.Error("killAwolServersComputeImage> error while deleting new image %s", imageID)
			}
		}

		if oldImageID != "" {
			log.Info("killAwolServersComputeImage> deleting old image for %s with ID %s", workerModelName, oldImageID)
			if err := images.Delete(h.openstackClient, oldImageID).ExtractErr(); err != nil {
				log.Error("killAwolServersComputeImage> error while deleting old image %s", oldImageID)
			}
		}
		h.resetImagesCache()
	}
}

func (h *HatcheryOpenstack) killErrorServers() {
	for _, s := range h.getServers() {
		//Remove server without IP Address
		if s.Status == "ACTIVE" {
			if len(s.Addresses) == 0 && time.Since(s.Updated) > 10*time.Minute {
				log.Info("Deleting server %s without IP Address", s.Name)

				r := servers.Delete(h.openstackClient, s.ID)
				if err := r.ExtractErr(); err != nil {
					log.Warning("killErrorServers> Cannot remove worker %s: %s", s.Name, err)
					continue
				}
			}
		}

		//Remove Error server
		if s.Status == "ERROR" {
			log.Info("Deleting server %s in error", s.Name)

			r := servers.Delete(h.openstackClient, s.ID)
			if err := r.ExtractErr(); err != nil {
				log.Warning("killErrorServers> Cannot remove worker in error %s: %s", s.Name, err)
				continue
			}
		}
	}
}

func (h *HatcheryOpenstack) killDisabledWorkers() {
	workers, err := h.Client().WorkerList()
	if err != nil {
		log.Warning("killDisabledWorkers> Cannot fetch worker list: %s", err)
		return
	}

	srvs := h.getServers()

	for _, w := range workers {
		if w.Status != sdk.StatusDisabled {
			continue
		}

		for _, s := range srvs {
			if s.Name == w.Name {
				log.Info("Deleting disabled worker %s", w.Name)
				r := servers.Delete(h.openstackClient, s.ID)
				if err := r.ExtractErr(); err != nil {
					log.Warning("killDisabledWorkers> Cannot disabled worker %s: %s", s.Name, err)
					continue
				}
			}
		}
	}
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryOpenstack) WorkersStarted() int {
	return len(h.getServers())
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryOpenstack) WorkersStartedByModel(model *sdk.Model) int {
	var x int
	for _, s := range h.getServers() {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(model.Name)) {
			x++
		}
	}
	log.Debug("WorkersStartedByModel> %s : %d", model.Name, x)
	return x
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryOpenstack) NeedRegistration(m *sdk.Model) bool {
	var oldDateLastModified string
	for _, img := range h.getImages() {
		w, _ := img.Metadata["worker_model_name"]
		if w == m.Name {
			if d, ok := img.Metadata["worker_model_last_modified"]; ok {
				oldDateLastModified = d.(string)
				break
			}
		}
	}

	var out bool
	if m.NeedRegistration || fmt.Sprintf("%d", m.UserLastModified.Unix()) != oldDateLastModified {
		out = true
	}
	log.Debug("NeedRegistration> %t for %s - m.NeedRegistration:%t m.UserLastModified:%d oldDateLastModified:%s", out, m.Name, m.NeedRegistration, m.UserLastModified.Unix(), oldDateLastModified)
	return out
}
