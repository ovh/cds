package vsphere

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

var hatcheryVSphere *HatcheryVSphere

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	hatch      *sdk.Hatchery
	images     []string
	datacenter *object.Datacenter
	finder     *find.Finder
	network    object.NetworkReference
	vclient    *govmomi.Client
	client     cdsclient.Interface

	// User provided parameters
	endpoint           string
	user               string
	password           string
	host               string
	datacenterString   string
	datastoreString    string
	networkString      string
	cardName           string
	workerTTL          int
	disableCreateImage bool
	createImageTimeout int
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryVSphere) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	for _, r := range requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	return true
}

//Client returns cdsclient instance
func (h *HatcheryVSphere) Client() cdsclient.Interface {
	return h.client
}

// KillWorker delete cloud instances
func (h *HatcheryVSphere) KillWorker(worker sdk.Worker) error {
	log.Info("KillWorker> Kill %s", worker.Name)
	for _, s := range h.getServers() {
		if s.Name == worker.Name {
			return h.deleteServer(s)
		}
	}
	return fmt.Errorf("not found")
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryVSphere) NeedRegistration(m *sdk.Model) bool {
	model, errG := h.getModelByName(m.Name)
	if errG != nil || model.Config == nil || model.Config.Annotation == "" {
		return true
	}

	var annot annotation
	if err := json.Unmarshal([]byte(model.Config.Annotation), &annot); err != nil {
		return true
	}

	return !annot.ToDelete && (m.NeedRegistration || m.UserLastModified.Unix() != annot.WorkerModelLastModified.Unix())
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryVSphere) WorkersStartedByModel(model *sdk.Model) int {
	var x int
	for _, s := range h.getServers() {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(model.Name)) {
			x++
		}
	}
	log.Debug("WorkersStartedByModel> %s : %d", model.Name, x)

	return x
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryVSphere) WorkersStarted() int {
	var x int
	for _, s := range h.getServers() {
		if strings.Contains(strings.ToLower(s.Name), "worker") {
			x++
		}
	}
	return x
}

//Hatchery returns hatchery instance
func (h *HatcheryVSphere) Hatchery() *sdk.Hatchery {
	return h.hatch
}

// ModelType returns type of hatchery
func (*HatcheryVSphere) ModelType() string {
	return sdk.VSphere
}

// ID returns hatchery id
func (h *HatcheryVSphere) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

func (h *HatcheryVSphere) main() {
	serverListTick := time.NewTicker(10 * time.Second).C
	killAwolServersTick := time.NewTicker(20 * time.Second).C
	killErrorServersTick := time.NewTicker(120 * time.Second).C
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

func (h *HatcheryVSphere) updateServerList() {
	var out string
	var total int
	status := map[string]int{}

	for _, s := range h.getServers() {
		out += fmt.Sprintf("- [%s] %s ", s.Summary.Config.Name, s.Summary.Runtime.PowerState)
		if _, ok := status[string(s.Summary.Runtime.PowerState)]; !ok {
			status[string(s.Summary.Runtime.PowerState)] = 0
		}
		status[string(s.Summary.Runtime.PowerState)]++
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

// killDisabledWorkers kill workers which are disabled
func (h *HatcheryVSphere) killDisabledWorkers() {
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
				if err := h.deleteServer(s); err != nil {
					log.Warning("killDisabledWorkers> Cannot disabled worker %s: %s", s.Name, err)
					continue
				}
			}
		}
	}
}

// killAwolServers kill unused servers
func (h *HatcheryVSphere) killAwolServers() {
	srvs := h.getServers()

	for _, s := range srvs {
		var annot annotation
		if s.Config == nil || s.Config.Annotation == "" {
			continue
		}
		if err := json.Unmarshal([]byte(s.Config.Annotation), &annot); err != nil {
			continue
		}

		if annot.ToDelete || (s.Summary.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn && (!annot.Model || annot.RegisterOnly)) {
			if err := h.deleteServer(s); err != nil {
				log.Warning("killAwolServers> cannot delete server %s", s.Name)
			}
		}
	}
}

// killErrorServers kill servers which are running for too long time
func (h *HatcheryVSphere) killErrorServers() {
	srvs := h.getServers()

	for _, s := range srvs {
		var annot annotation
		if s.Config == nil || s.Config.Annotation == "" {
			continue
		}
		if err := json.Unmarshal([]byte(s.Config.Annotation), &annot); err != nil {
			continue
		}

		if !annot.Model && annot.Created.Before(time.Now().Add(-2*time.Hour)) {
			if err := h.deleteServer(s); err != nil {
				log.Warning("killErrorServers> cannot delete server %s", s.Name)
			}
		}
	}
}
