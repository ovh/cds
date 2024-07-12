package vcs

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/vcs/bitbucketcloud"
	"github.com/ovh/cds/engine/vcs/bitbucketserver"
	"github.com/ovh/cds/engine/vcs/gerrit"
	"github.com/ovh/cds/engine/vcs/gitea"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/engine/vcs/gitlab"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines(context.Background())
	return s
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid vcs configuration"))
	}
	s.Router = &api.Router{
		Mux:    mux.NewRouter(),
		Config: sConfig.HTTP,
	}
	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
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

	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeVCS
	s.HTTPURL = s.Cfg.URL
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

func (s *Service) getConsumer(name string, vcsAuth sdk.VCSAuth) (sdk.VCSServer, error) {
	switch vcsAuth.Type {
	case sdk.VCSTypeGitea:
		return gitea.New(strings.TrimSuffix(vcsAuth.URL, "/"),
			s.Cfg.API.HTTP.URL,
			s.UI.HTTP.URL,
			s.Cfg.ProxyWebhook,
			s.Cache,
			vcsAuth.Username,
			vcsAuth.Token,
		), nil
	case sdk.VCSTypeBitbucketCloud:
		return bitbucketcloud.New(
			strings.TrimSuffix(vcsAuth.URL, "/"),
			s.UI.HTTP.URL,
			s.Cfg.ProxyWebhook,
			s.Cache,
		), nil
	case sdk.VCSTypeBitbucketServer:
		return bitbucketserver.New(
			strings.TrimSuffix(vcsAuth.URL, "/"),
			s.Cfg.API.HTTP.URL,
			s.UI.HTTP.URL,
			s.Cfg.ProxyWebhook,
			s.Cache,
			vcsAuth.Username,
			vcsAuth.Token,
		), nil
	case sdk.VCSTypeGerrit:
		return gerrit.New(
			vcsAuth.URL,
			s.Cache,
			vcsAuth.SSHUsername,
			vcsAuth.SSHPort,
			vcsAuth.Username,
			vcsAuth.Token,
		), nil
	case sdk.VCSTypeGithub:
		return github.New(
			vcsAuth.URL,
			vcsAuth.URLApi,
			s.Cfg.API.HTTP.URL,
			s.UI.HTTP.URL,
			s.Cfg.ProxyWebhook,
			s.Cache,
		), nil
	case sdk.VCSTypeGitlab:
		return gitlab.New(
			vcsAuth.URL,
			s.UI.HTTP.URL,
			s.Cfg.ProxyWebhook,
			s.Cache,
			vcsAuth.Username,
			vcsAuth.Token,
		), nil
	}
	return nil, sdk.WithStack(sdk.ErrNotFound)
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	log.Info(c, "VCS> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	// Retrieve UI URL from API
	cfgUser, err := s.Client.ConfigUser()
	if err != nil {
		return err
	}
	s.UI.HTTP.URL = cfgUser.URLUI

	s.Cache, err = cache.New(c, s.Cfg.Cache.Redis, s.Cfg.Cache.TTL)
	if err != nil {
		return fmt.Errorf("Cannot connect to redis instance : %v", err)
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
	log.Info(c, "VCS> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error(c, "VCS> Listen and serve failed: %s", err)
	}

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-c.Done():
			log.Info(c, "VCS> Shutdown HTTP Server")
			server.Shutdown(c)
		}
	}()

	return c.Err()
}
