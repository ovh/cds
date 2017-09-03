package openstack

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

var hatcheryOpenStack *HatcheryOpenstack

var workersAlive map[string]int64

type ipInfos struct {
	workerName     string
	dateLastBooked time.Time
}

var ipsInfos = struct {
	mu  sync.RWMutex
	ips map[string]ipInfos
}{
	mu:  sync.RWMutex{},
	ips: map[string]ipInfos{},
}

// HatcheryOpenstack spawns instances of worker model with type 'ISO'
// by startup up virtual machines on /cloud
type HatcheryOpenstack struct {
	hatch           *sdk.Hatchery
	flavors         []flavors.Flavor
	networks        []tenantnetworks.Network
	images          []images.Image
	openstackClient *gophercloud.ServiceClient
	client          cdsclient.Interface

	// User provided parameters
	address            string
	user               string
	password           string
	endpoint           string
	tenant             string
	region             string
	networkString      string // flag from cli
	networkID          string // computed from networkString
	workerTTL          int
	disableCreateImage bool
	createImageTimeout int
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
			if !h.disableCreateImage && s.Status == "SHUTOFF" && registerOnly == "true" {
				h.killAwolServersComputeImage(workerModelName, workerModelNameLastModified, s.ID, model, flavor)
			}

			log.Info("killAwolServers> Deleting server %s status: %s last update: %s registerOnly:%s toDeleteKilled:%t inWorkersList:%t", s.Name, s.Status, time.Since(s.Updated), registerOnly, toDeleteKilled, inWorkersList)
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
		log.Info("killAwolServersComputeImage> image %s created for worker model %s - waiting %ds for saving created img...", imageID, workerModelName, h.createImageTimeout)

		startTime := time.Now().Unix()
		var newImageIsActive bool
		for time.Now().Unix()-startTime < int64(hatcheryOpenStack.createImageTimeout) {
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
	log.Info("NeedRegistration> %t for %s - m.NeedRegistration:%t m.UserLastModified:%d oldDateLastModified:%s", out, m.Name, m.NeedRegistration, m.UserLastModified.Unix(), oldDateLastModified)
	return out
}

// KillWorker delete cloud instances
func (h *HatcheryOpenstack) KillWorker(worker sdk.Worker) error {
	log.Info("KillWorker> Kill %s", worker.Name)
	for _, s := range h.getServers() {
		if s.Name == worker.Name {
			r := servers.Delete(h.openstackClient, s.ID)
			if err := r.ExtractErr(); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("not found")
}
