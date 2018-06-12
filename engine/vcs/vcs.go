package vcs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/vcs/bitbucket"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/engine/vcs/gitlab"
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

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeVCS
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

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

func (s *Service) getConsumer(name string) (sdk.VCSServer, error) {
	serverCfg, has := s.Cfg.Servers[name]
	if !has {
		return nil, sdk.ErrNotFound
	}
	fmt.Printf("%+v", serverCfg.Github)
	if serverCfg.Github != nil {
		return github.New(
			serverCfg.Github.ClientID,
			serverCfg.Github.ClientSecret,
			s.Cfg.API.HTTP.URL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.Github.ProxyWebhook,
			s.Cache,
			serverCfg.Github.Status.Disable,
			!serverCfg.Github.Status.ShowDetail,
		), nil
	}
	if serverCfg.Bitbucket != nil {
		return bitbucket.New(serverCfg.Bitbucket.ConsumerKey,
			[]byte(serverCfg.Bitbucket.PrivateKey),
			serverCfg.URL,
			s.Cfg.API.HTTP.URL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.Bitbucket.ProxyWebhook,
			s.Cache,
			serverCfg.Bitbucket.Status.Disable,
		), nil
	}
	if serverCfg.Gitlab != nil {
		return gitlab.New(serverCfg.Gitlab.AppID,
			serverCfg.Gitlab.Secret,
			serverCfg.URL,
			s.Cfg.API.HTTP.URL+"/repositories_manager/oauth2/callback",
			s.Cfg.UI.HTTP.URL,
			serverCfg.Gitlab.ProxyWebhook,
			s.Cache,
			serverCfg.Gitlab.Status.Disable,
			serverCfg.Gitlab.Status.ShowDetail,
		), nil
	}
	return nil, sdk.ErrNotFound
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	log.Info("VCS> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	//Init the cache
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("Cannot connect to redis instance : %v", errCache)
	}

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
	log.Info("VCS> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("VCS> Listen and serve failed: %s", err)
	}

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-c.Done():
			log.Info("VCS> Shutdown HTTP Server")
			server.Shutdown(c)
		}
	}()

	return c.Err()
}
