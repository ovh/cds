package vsphere

import (
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

var hatcheryVSphere *HatcheryVSphere
var workersAlive map[string]int64

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	hatch      *sdk.Hatchery
	images     []string
	datacenter *object.Datacenter
	finder     *find.Finder
	network    object.NetworkReference
	client     *govmomi.Client

	// User provided parameters
	endpoint           string
	user               string
	password           string
	host               string
	datacenterString   string
	datastoreString    string
	networkString      string
	workerTTL          int
	disableCreateImage bool
	createImageTimeout int
}

// CanSpawn return wether or not hatchery can spawn model
// requirements are not supported
func (h *HatcheryVSphere) CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool {
	for _, r := range job.Job.Action.Requirements {
		if r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			return false
		}
	}
	return true
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
	// Laucnh worker with register and create vm if it doesn't exist and let them alive
	return true
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
