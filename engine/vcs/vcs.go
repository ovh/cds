package vcs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/vcs/bitbucketcloud"
	"github.com/ovh/cds/engine/vcs/bitbucketserver"
	"github.com/ovh/cds/engine/vcs/gerrit"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/engine/vcs/gitlab"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid vcs configuration"))
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
	s.ServiceName = "cds-vcs"

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
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	if serverCfg.Github != nil {
		return github.New(
			serverCfg.Github.ClientID,
			serverCfg.Github.ClientSecret,
			serverCfg.URL,
			serverCfg.Github.APIURL,
			s.Cfg.API.HTTP.URL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.Github.ProxyWebhook,
			serverCfg.Github.Username,
			serverCfg.Github.Token,
			s.Cache,
			serverCfg.Github.Status.Disable,
			!serverCfg.Github.Status.ShowDetail,
		), nil
	}
	if serverCfg.Bitbucket != nil {
		return bitbucketserver.New(serverCfg.Bitbucket.ConsumerKey,
			[]byte(serverCfg.Bitbucket.PrivateKey),
			serverCfg.URL,
			s.Cfg.API.HTTP.URL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.Bitbucket.ProxyWebhook,
			serverCfg.Bitbucket.Username,
			serverCfg.Bitbucket.Token,
			s.Cache,
			serverCfg.Bitbucket.Status.Disable,
		), nil
	}
	if serverCfg.BitbucketCloud != nil {
		return bitbucketcloud.New(serverCfg.BitbucketCloud.ClientID,
			serverCfg.BitbucketCloud.ClientSecret,
			serverCfg.URL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.BitbucketCloud.ProxyWebhook,
			s.Cache,
			serverCfg.BitbucketCloud.Status.Disable,
			!serverCfg.BitbucketCloud.Status.ShowDetail,
		), nil
	}
	if serverCfg.Gitlab != nil {
		return gitlab.New(serverCfg.Gitlab.AppID,
			serverCfg.Gitlab.Secret,
			serverCfg.URL,
			serverCfg.Gitlab.CallbackURL,
			s.Cfg.UI.HTTP.URL,
			serverCfg.Gitlab.ProxyWebhook,
			s.Cache,
			serverCfg.Gitlab.Status.Disable,
			serverCfg.Gitlab.Status.ShowDetail,
		), nil
	}
	if serverCfg.Gerrit != nil {
		return gerrit.New(
			serverCfg.URL,
			s.Cache,
			serverCfg.Gerrit.Status.Disable,
			serverCfg.Gerrit.Status.ShowDetail,
			serverCfg.Gerrit.SSHPort,
			serverCfg.Gerrit.Reviewer.User,
			serverCfg.Gerrit.Reviewer.Token), nil
	}
	return nil, sdk.WithStack(sdk.ErrNotFound)
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	log.Info(c, "VCS> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
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
