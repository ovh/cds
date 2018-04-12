package vcs

import (
	"fmt"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Service is the stuct representing a vcs µService
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS VCS Service"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8084" toml:"name"`
	} `toml:"http" comment:"######################\n CDS VCS HTTP Configuration \n######################"`
	URL string `default:"http://localhost:8084"`
	UI  struct {
		HTTP struct {
			URL string `toml:"url" default:"http://localhost:2015"`
		} `toml:"http"`
	}
	API struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081"`
			Insecure bool   `toml:"insecure" commented:"true"`
		} `toml:"http"`
		GRPC struct {
			URL      string `toml:"url" default:"http://localhost:8082"`
			Insecure bool   `toml:"insecure" commented:"true"`
		} `toml:"grpc"`
		Token                string `toml:"token" default:"************"`
		RequestTimeout       int    `toml:"requestTimeout" default:"10"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10"`
	} `toml:"api" comment:"######################\n CDS API Settings \n######################"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################"`
	Servers map[string]ServerConfiguration `toml:"servers" comment:"######################\n CDS VCS Server Settings \n######################"`
}

// ServerConfiguration is the configuration for a VCS server
type ServerConfiguration struct {
	URL       string                        `toml:"url" comment:"URL of this VCS Server" json:"url"`
	Github    *GithubServerConfiguration    `toml:"github" json:"github,omitempty"`
	Gitlab    *GitlabServerConfiguration    `toml:"gitlab" json:"gitlab,omitempty"`
	Bitbucket *BitbucketServerConfiguration `toml:"bitbucket" json:"bitbucket,omitempty"`
}

// GithubServerConfiguration represents the github configuration
type GithubServerConfiguration struct {
	ClientID     string `toml:"clientId" json:"-" comment:"Github OAuth Application Client ID"`
	ClientSecret string `toml:"clientSecret" json:"-"  comment:"Github OAuth Application Client Secret"`
	Status       struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableWebHooks bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
}

func (s GithubServerConfiguration) check() error {
	if s.ClientID == "" {
		return errGithubConfigurationError
	}
	if s.ClientSecret == "" {
		return errGithubConfigurationError
	}
	return nil
}

var errGithubConfigurationError = fmt.Errorf("Github configuration Error")

// GitlabServerConfiguration represents the gitlab configuration
type GitlabServerConfiguration struct {
	AppID  string `toml:"appId" json:"-"`
	Secret string `toml:"secret" json:"-"`
	Status struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableWebHooks bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
}

func (s GitlabServerConfiguration) check() error {
	return nil
}

// BitbucketServerConfiguration represents the bitbucket configuration
type BitbucketServerConfiguration struct {
	ConsumerKey string `toml:"consumerKey" json:"-" comment:"You can change the consumeKey if you want"`
	PrivateKey  string `toml:"privateKey" json:"-"`
	Status      struct {
		Disable bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
	}
	DisableWebHooks bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
}

func (s BitbucketServerConfiguration) check() error {
	return nil
}

func (s *Service) addServerConfiguration(name string, c ServerConfiguration) error {
	if name == "" {
		return fmt.Errorf("Invalid VCS server name")
	}

	if err := c.check(); err != nil {
		return sdk.WrapError(err, "Unable to add server configuration")
	}
	if s.Cfg.Servers == nil {
		s.Cfg.Servers = map[string]ServerConfiguration{}
	}
	s.Cfg.Servers[name] = c
	log.Debug("VCS> addServerConfiguration %+v %+v", s.Cfg.Servers[name], s.Cfg.Servers[name].Github)
	return nil
}

func (s ServerConfiguration) check() error {
	if s.URL == "" {
		return fmt.Errorf("Invalid VCS server URL")
	}

	if s.Bitbucket != nil && s.Github != nil && s.Gitlab != nil {
		return fmt.Errorf("Invalid VCS server configuration")
	}

	if s.Bitbucket != nil {
		if err := s.Bitbucket.check(); err != nil {
			return err
		}
	}

	if s.Github != nil {
		if err := s.Github.check(); err != nil {
			return err
		}
	}

	if s.Gitlab != nil {
		if err := s.Gitlab.check(); err != nil {
			return err
		}
	}

	return nil
}
