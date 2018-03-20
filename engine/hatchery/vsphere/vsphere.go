package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// New instanciates a new Hatchery vsphere
func New() *HatcheryVSphere {
	return new(HatcheryVSphere)
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryVSphere) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryVSphere) CheckConfiguration(cfg interface{}) error {
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

	if hconfig.VSphereUser == "" {
		return fmt.Errorf("vsphere-user is mandatory")
	}

	if hconfig.VSphereEndpoint == "" {
		return fmt.Errorf("vsphere-endpoint is mandatory")
	}

	if hconfig.VSpherePassword == "" {
		return fmt.Errorf("vsphere-password is mandatory")
	}

	if hconfig.VSphereDatacenterString == "" {
		return fmt.Errorf("vsphere-datacenter is mandatory")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your vsphere hatchery configuration")
	}

	return nil
}

// Serve start the HatcheryVSphere server
func (h *HatcheryVSphere) Serve(ctx context.Context) error {
	return hatchery.Create(h)
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

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryVSphere) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
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

	return !annot.ToDelete && (m.NeedRegistration || fmt.Sprintf("%d", m.UserLastModified.Unix()) != annot.WorkerModelLastModified)
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
	killDisabledWorkersTick := time.NewTicker(60 * time.Second).C

	for {
		select {

		case <-serverListTick:
			h.updateServerList()

		case <-killAwolServersTick:
			h.killAwolServers()

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
