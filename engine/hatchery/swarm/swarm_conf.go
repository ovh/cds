package swarm

import (
	"context"
	"fmt"
	"time"

	types "github.com/docker/docker/api/types"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func (s *HatcherySwarm) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid swarm hatchery configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

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
		RatioService: &h.Config.RatioService,
	}

	h.Name = h.Config.Name
	h.HTTPURL = h.Config.URL
	h.Type = services.TypeHatchery
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.Common.Common.ServiceName = "cds-hatchery-swarm"

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcherySwarm) Status() sdk.MonitoringStatus {
	m := h.CommonMonitoring()
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted()), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	for dockerName, dockerClient := range h.dockerClients {
		//Check images
		status := sdk.MonitoringStatusOK
		ctxList, cancelList := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancelList()
		images, err := dockerClient.ImageList(ctxList, types.ImageListOptions{All: true})
		if err != nil {
			log.Warning("hatchery> swarm> %s> Status> Unable to list images on %s: %s", h.Name, dockerName, err)
			status = sdk.MonitoringStatusAlert
		}
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Images-" + dockerName, Value: fmt.Sprintf("%d", len(images)), Status: status})
		//Check containers
		status = sdk.MonitoringStatusOK
		cs, err := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			log.Warning("hatchery> swarm> %s> Status> Unable to list containers on %s: %s", h.Name, dockerName, err)
			status = sdk.MonitoringStatusAlert
		}
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Containers-" + dockerName, Value: fmt.Sprintf("%d", len(cs)), Status: status})
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

	if hconfig.WorkerTTL <= 0 {
		return fmt.Errorf("worker-ttl must be > 0")
	}
	if hconfig.DefaultMemory <= 1 {
		return fmt.Errorf("worker-memory must be > 1")
	}

	if hconfig.Name == "" {
		return fmt.Errorf("please enter a name in your swarm hatchery configuration")
	}

	return nil
}
