package main

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// New instanciates a new hatchery local
func New() *HatcheryKubernetes {
	return new(HatcheryKubernetes)
}

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcheryKubernetes) ApplyConfiguration(cfg interface{}) error {
	// if err := h.CheckConfiguration(cfg); err != nil {
	// 	return err
	// }
	//
	// var ok bool
	// h.Config, ok = cfg.(HatcheryConfiguration)
	// if !ok {
	// 	return fmt.Errorf("Invalid configuration")
	// }

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcheryKubernetes) CheckConfiguration(cfg interface{}) error {
	// hconfig, ok := cfg.(HatcheryConfiguration)
	// if !ok {
	// 	return fmt.Errorf("Invalid configuration")
	// }
	//
	// if hconfig.API.HTTP.URL == "" {
	// 	return fmt.Errorf("API HTTP(s) URL is mandatory")
	// }
	//
	// if hconfig.API.Token == "" {
	// 	return fmt.Errorf("API Token URL is mandatory")
	// }
	//
	// if hconfig.Basedir == "" {
	// 	return fmt.Errorf("Invalid basedir directory")
	// }
	//
	// if hconfig.Name == "" {
	// 	return fmt.Errorf("please enter a name in your local hatchery configuration")
	// }
	//
	// if ok, err := api.DirectoryExists(hconfig.Basedir); !ok {
	// 	return fmt.Errorf("Basedir doesn't exist")
	// } else if err != nil {
	// 	return fmt.Errorf("Invalid basedir: %v", err)
	// }
	return nil
}

// Serve start the HatcheryKubernetes server
func (h *HatcheryKubernetes) Serve(ctx context.Context) error {
	hatchery.Create(h)
	return nil
}

// ID must returns hatchery id
func (h *HatcheryKubernetes) ID() int64 {
	if h.hatch == nil {
		return 0
	}
	return h.hatch.ID
}

//Hatchery returns hatchery instance
func (h *HatcheryKubernetes) Hatchery() *sdk.Hatchery {
	return h.hatch
}

//Client returns cdsclient instance
func (h *HatcheryKubernetes) Client() cdsclient.Interface {
	return h.client
}

//Configuration returns Hatchery CommonConfiguration
func (h *HatcheryKubernetes) Configuration() hatchery.CommonConfiguration {
	return h.Config.CommonConfiguration
}

// ModelType returns type of hatchery
func (*HatcheryKubernetes) ModelType() string {
	return sdk.HostProcess
}

// CanSpawn return wether or not hatchery can spawn model.
// requirements are not supported
func (h *HatcheryKubernetes) CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool {
	return true
}

// killWorker kill a local process
func (h *HatcheryKubernetes) killWorker(worker sdk.Worker) error {
	return nil
}

// SpawnWorker starts a new worker process
func (h *HatcheryKubernetes) SpawnWorker(spawnArgs hatchery.SpawnArguments) (string, error) {

	return "wName", nil
}

// WorkersStarted returns the number of instances started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStarted() int {
	return len(h.workers)
}

// WorkersStartedByModel returns the number of instances of given model started but
// not necessarily register on CDS yet
func (h *HatcheryKubernetes) WorkersStartedByModel(model *sdk.Model) int {
	return 0
}

// checkCapabilities checks all requirements, foreach type binary, check if binary is on current host
// returns an error "Exit status X" if current host misses one requirement
func checkCapabilities(req []sdk.Requirement) ([]sdk.Requirement, error) {
	var capa []sdk.Requirement

	return capa, nil
}

// Init register local hatchery with its worker model
func (h *HatcheryKubernetes) Init() error {
	h.workers = make(map[string]workerCmd)

	genname := h.Configuration().Name
	h.client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		genname,
	)

	req, err := h.Client().Requirements()
	if err != nil {
		return fmt.Errorf("Cannot fetch requirements: %s", err)
	}

	capa, err := checkCapabilities(req)
	if err != nil {
		return fmt.Errorf("Cannot check local capabilities: %s", err)
	}

	h.hatch = &sdk.Hatchery{
		Name: genname,
		Model: sdk.Model{
			Name:         genname,
			Image:        genname,
			Capabilities: capa,
			Provision:    int64(h.Config.NbProvision),
		},
		Version: sdk.VERSION,
	}

	if err := hatchery.Register(h); err != nil {
		return fmt.Errorf("Cannot register: %s", err)
	}

	go h.startKillAwolWorkerRoutine()
	return nil
}

func (h *HatcheryKubernetes) localWorkerIndexCleanup() {

}

func (h *HatcheryKubernetes) startKillAwolWorkerRoutine() {

}

func (h *HatcheryKubernetes) killAwolWorkers() error {

	return nil
}

// NeedRegistration return true if worker model need regsitration
func (h *HatcheryKubernetes) NeedRegistration(m *sdk.Model) bool {
	if m.NeedRegistration || m.LastRegistration.Unix() < m.UserLastModified.Unix() {
		return true
	}
	return false
}
