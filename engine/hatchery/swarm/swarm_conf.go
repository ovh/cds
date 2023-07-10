package swarm

import (
	"context"
	"fmt"
	"time"

	types "github.com/docker/docker/api/types"
	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Init initializes the swarm hatchery
func (h *HatcherySwarm) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(HatcheryConfiguration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid swarm hatchery configuration"))
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
	h.Common.Common.Region = h.Config.Provision.Region
	h.Common.Common.IgnoreJobWithNoRegion = h.Config.Provision.IgnoreJobWithNoRegion
	h.Common.Common.ModelType = h.ModelType()

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (h *HatcherySwarm) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := h.NewMonitoringStatus()
	ws, err := h.WorkersStarted(ctx)
	if err != nil {
		ctx = log.ContextWithStackTrace(ctx, err)
		log.Warn(ctx, err.Error())
	}
	m.AddLine(sdk.MonitoringStatusLine{Component: "Workers", Value: fmt.Sprintf("%d/%d", len(ws), h.Config.Provision.MaxWorker), Status: sdk.MonitoringStatusOK})
	var nbErrorImageList, nbErrorGetContainers int
	for dockerName, dockerClient := range h.dockerClients {
		//Check images
		status := sdk.MonitoringStatusOK
		ctxList, cancelList := context.WithTimeout(ctx, 20*time.Second)
		defer cancelList()
		images, err := dockerClient.ImageList(ctxList, types.ImageListOptions{All: true})
		if err != nil {
			log.Warn(ctx, "hatchery> swarm> %s> Status> Unable to list images on %s: %s", h.Name(), dockerName, err)
			status = sdk.MonitoringStatusWarn
			nbErrorImageList++
		}
		m.AddLine(sdk.MonitoringStatusLine{Component: "Images-" + dockerName, Value: fmt.Sprintf("%d", len(images)), Status: status})
		//Check containers
		status = sdk.MonitoringStatusOK
		cs, err := h.getContainers(ctx, dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			log.Warn(ctx, "hatchery> swarm> %s> Status> Unable to list containers on %s: %s", h.Name(), dockerName, err)
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

	if hconfig.DefaultMemory <= 1 {
		return fmt.Errorf("worker-memory must be > 1")
	}

	return nil
}
