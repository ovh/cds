package ui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

// ApplyConfiguration apply an object of type hooks.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second, s.Cfg.API.HTTP.Insecure)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeUI
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures
	s.ServiceName = "cds-ui"

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your vcs configuration")
	}

	return nil
}

// Serve will start the http ui server
func (s *Service) Serve(c context.Context) error {
	log.Info("UI> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	//Init the http server
	s.initRouter(c)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	//Start the http server
	log.Info("UI> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("UI> Listen and serve failed: %s", err)
	}

	//Gracefully shutdown the http server
	<-c.Done()
	log.Info("UI> Shutdown HTTP Server")
	if err := server.Shutdown(c); err != nil {
		return fmt.Errorf("unable to shutdown server: %v", err)
	}

	return c.Err()
}
