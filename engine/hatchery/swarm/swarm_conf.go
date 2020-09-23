package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	types "github.com/docker/docker/api/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// Init initializes the swarm hatchery
func (h *HatcherySwarm) Init(config interface{}) (cdsclient.ServiceConfig, error) {
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

	h.HTTPURL = h.Config.URL
	h.MaxHeartbeatFailures = h.Config.API.MaxHeartbeatFailures
	h.Common.Common.ServiceName = h.Config.Name
	h.Common.Common.ServiceType = sdk.TypeHatchery
	var err error
	h.Common.Common.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Config.RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcherySwarm) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(h.WorkersStarted(ctx)), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	var nbErrorImageList, nbErrorGetContainers int
	for dockerName, dockerClient := range h.dockerClients {
		//Check images
		status := sdk.MonitoringStatusOK
		ctxList, cancelList := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancelList()
		images, err := dockerClient.ImageList(ctxList, types.ImageListOptions{All: true})
		if err != nil {
			log.Warning(ctx, "hatchery> swarm> %s> Status> Unable to list images on %s: %s", h.Name(), dockerName, err)
			status = sdk.MonitoringStatusWarn
			nbErrorImageList++
		}
		m.AddLine(sdk.MonitoringStatusLine{Component: "Images-" + dockerName, Value: fmt.Sprintf("%d", len(images)), Status: status})
		//Check containers
		status = sdk.MonitoringStatusOK
		cs, err := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			log.Warning(ctx, "hatchery> swarm> %s> Status> Unable to list containers on %s: %s", h.Name(), dockerName, err)
			status = sdk.MonitoringStatusWarn
			nbErrorGetContainers++
		}
		m.AddLine(sdk.MonitoringStatusLine{Component: "Containers-" + dockerName, Value: fmt.Sprintf("%d", len(cs)), Status: status})
	}

	var status = sdk.MonitoringStatusOK
	if nbErrorImageList > len(h.dockerClients)/2 {
		status = sdk.MonitoringStatusAlert
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "DockerEngines.ListImages", Value: fmt.Sprintf("%d/%d", nbErrorImageList, len(h.dockerClients)), Status: status})

	status = sdk.MonitoringStatusOK
	if nbErrorGetContainers > len(h.dockerClients)/2 {
		status = sdk.MonitoringStatusAlert
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "DockerEngines.GetContainers", Value: fmt.Sprintf("%d/%d", nbErrorGetContainers, len(h.dockerClients)), Status: status})

	return m
}

// CheckConfiguration checks the validity of the configuration object
func (h *HatcherySwarm) CheckConfiguration(cfg interface{}) error {
	hconfig, ok := cfg.(HatcheryConfiguration)
	if !ok {
		return fmt.Errorf("Invalid hatchery swarm configuration")
	}

	if err := hconfig.Check(); err != nil {
		return fmt.Errorf("Invalid hatchery swarm configuration: %v", err)
	}

	if hconfig.WorkerTTL <= 0 {
		return fmt.Errorf("worker-ttl must be > 0")
	}
	if hconfig.DefaultMemory <= 1 {
		return fmt.Errorf("worker-memory must be > 1")
	}

	return nil
}
