package swarm

import (
	"fmt"
	"os"
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

	// s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second)
	// s.API = s.Cfg.API.HTTP.URL
	// s.Name = s.Cfg.Name
	// s.HTTPURL = s.Cfg.URL
	// s.Token = s.Cfg.API.Token
	// s.Type = services.TypeHatchery
	// s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	return nil
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
