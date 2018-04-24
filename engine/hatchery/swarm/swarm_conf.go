package swarm

import (
	"context"
	"fmt"
	"os"

	types "github.com/docker/docker/api/types"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// ApplyConfiguration apply an object of type HatcheryConfiguration after checking it
func (h *HatcherySwarm) ApplyConfiguration(cfg interface{}) error {
	if err := h.CheckConfiguration(cfg); err != nil {
		return err
	}

	var ok bool
	h.Config, ok = cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	h.hatch = &sdk.Hatchery{
		Name:    h.Configuration().Name,
		Version: sdk.VERSION,
	}

	h.Client = cdsclient.NewHatchery(
		h.Configuration().API.HTTP.URL,
		h.Configuration().API.Token,
		h.Configuration().Provision.RegisterFrequency,
		h.Configuration().API.HTTP.Insecure,
		h.hatch.Name,
	)

	h.API = h.Config.API.HTTP.URL
	h.Name = h.Config.Name
	h.HTTPURL = h.Config.URL
	h.Token = h.Config.API.Token
	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcherySwarm) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()

	if h.IsInitialized() {
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", h.WorkersStarted(), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})

		status := sdk.MonitoringStatusOK
		images, err := h.dockerClient.ImageList(context.Background(), types.ImageListOptions{All: true})
		if err != nil {
			log.Warning("%d> Status> Unable to list images: %s", h.Name, err)
			status = sdk.MonitoringStatusAlert
		}

		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Images", Value: fmt.Sprintf("%d", len(images)), Status: status})
	}

	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcherySwarm) CheckConfiguration(cfg interface{}) error {
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

	if hconfig.MaxContainers <= 0 {
		return fmt.Errorf("max-containers must be > 0")
	}
	if hconfig.WorkerTTL <= 0 {
		return fmt.Errorf("worker-ttl must be > 0")
	}
	if hconfig.DefaultMemory <= 1 {
		return fmt.Errorf("worker-memory must be > 1")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your swarm hatchery configuration")
	}

	if os.Getenv("DOCKER_HOST") == "" {
		return fmt.Errorf("Please export docker client env variables DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH")
	}

	return nil
}
