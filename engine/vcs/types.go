package vcs

import (
	"fmt"
	"strings"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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
	Name string `toml:"name" comment:"Name of this CDS VCS Service\n Enter a name to enable this service" json:"name"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8084" json:"port"`
	} `toml:"http" comment:"######################\n CDS VCS HTTP Configuration \n######################" json:"http"`
	URL string `default:"http://localhost:8084" json:"url"`
	UI  struct {
		HTTP struct {
			URL string `toml:"url" default:"http://localhost:2015" json:"url"`
		} `toml:"http" json:"http"`
	}
	API   service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################" json:"cache"`
	Servers map[string]ServerConfiguration `toml:"servers" comment:"######################\n CDS VCS Server Settings \n######################" json:"servers"`
}

// ServerConfiguration is the configuration for a VCS server
type ServerConfiguration struct {
	URL            string                        `toml:"url" comment:"URL of this VCS Server" json:"url"`
	Github         *GithubServerConfiguration    `toml:"github" json:"github,omitempty"`
	Gitlab         *GitlabServerConfiguration    `toml:"gitlab" json:"gitlab,omitempty"`
	Bitbucket      *BitbucketServerConfiguration `toml:"bitbucket" json:"bitbucket,omitempty"`
	BitbucketCloud *BitbucketCloudConfiguration  `toml:"bitbucketcloud" json:"bitbucketcloud,omitempty"`
	Gerrit         *GerritServerConfiguration    `toml:"gerrit" json:"gerrit,omitempty"`
}

// GithubServerConfiguration represents the github configuration
type GithubServerConfiguration struct {
	ClientID     string `toml:"clientId" json:"-" default:"xxxxx" comment:"#######\n CDS <-> Github. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/github/ \n#######\n Github OAuth Application Client ID"`
	ClientSecret string `toml:"clientSecret" json:"-" default:"xxxxx" comment:"Github OAuth Application Client Secret"`
	APIURL       string `toml:"apiUrl" json:"-" default:"https://api.github.com" comment:"The URL for the GitHub API."`
	Status       struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableWebHooks bool   `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool   `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
	ProxyWebhook    string `toml:"proxyWebhook" default:"https://myproxy.com" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
	Username        string `toml:"username" comment:"optional. Github username, used to add comment on Pull Request on failed build." json:"username"`
	Token           string `toml:"token" comment:"optional, Github Token associated to username, used to add comment on Pull Request" json:"-"`
}

func (s GithubServerConfiguration) check() error {
	if s.ClientID == "" {
		return errGithubConfigurationError
	}
	if s.ClientSecret == "" {
		return errGithubConfigurationError
	}
	if s.ProxyWebhook != "" && !strings.Contains(s.ProxyWebhook, "://") {
		return fmt.Errorf("Github proxy webhook must have the HTTP scheme")
	}
	return nil
}

var errGithubConfigurationError = fmt.Errorf("Github configuration Error")

// GitlabServerConfiguration represents the gitlab configuration
type GitlabServerConfiguration struct {
	AppID       string `toml:"appId" json:"-" default:"xxxxx" comment:"#######\n CDS <-> Gitlab. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/gitlab/ \n#######"`
	Secret      string `toml:"secret" json:"-" default:"xxxxx"`
	CallbackURL string `toml:"callbackUrl" json:"callbackUrl" default:"http://localhost:8081/repositories_manager/oauth2/callback" comment:"OAuth Application Callback URL"`
	Status      struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableWebHooks bool   `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool   `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
	ProxyWebhook    string `toml:"proxyWebhook" default:"https://myproxy.com" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
}

func (s GitlabServerConfiguration) check() error {
	if s.ProxyWebhook != "" && !strings.Contains(s.ProxyWebhook, "://") {
		return fmt.Errorf("Gitlab proxy webhook must have the HTTP scheme")
	}
	return nil
}

// BitbucketServerConfiguration represents the bitbucket configuration
type BitbucketServerConfiguration struct {
	ConsumerKey string `toml:"consumerKey" json:"-" default:"xxxxx" comment:"#######\n CDS <-> Bitbucket. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/bitbucket/ \n#######\n You can change the consumeKey if you want"`
	PrivateKey  string `toml:"privateKey" json:"-" default:"xxxxx"`
	Status      struct {
		Disable bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
	}
	DisableWebHooks bool   `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	DisablePolling  bool   `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
	ProxyWebhook    string `toml:"proxyWebhook" default:"https://myproxy.com" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
	Username        string `toml:"username" comment:"optional. Bitbucket username, used to add comment on Pull Request on failed build." json:"username"`
	Token           string `toml:"token" comment:"optional, Bitbucket Token associated to username, used to add comment on Pull Request" json:"-"`
}

func (s BitbucketServerConfiguration) check() error {
	if s.ProxyWebhook != "" && !strings.Contains(s.ProxyWebhook, "://") {
		return fmt.Errorf("Bitbucket proxy webhook must have the HTTP scheme")
	}
	return nil
}

// BitbucketCloudConfiguration represents the bitbucket configuration
type BitbucketCloudConfiguration struct {
	ClientID     string `toml:"clientId" json:"-" default:"xxxxx" comment:"#######\n CDS <-> Bitbucket cloud. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/bitbucketcloud/ \n#######\n Bitbucket cloud OAuth Application Client ID"`
	ClientSecret string `toml:"clientSecret" json:"-" default:"xxxxx" comment:"Bitbucket Cloud OAuth Application Client Secret"`
	Status       struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableWebHooks bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server" json:"disable_web_hook"`
	// DisablePolling  bool   `toml:"disablePolling" comment:"Does polling is supported by VCS Server" json:"disable_polling"`
	ProxyWebhook string `toml:"proxyWebhook" default:"https://myproxy.com" commented:"true" comment:"If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK" json:"proxy_webhook"`
	// Username        string `toml:"username" comment:"optional. Github username, used to add comment on Pull Request on failed build." json:"username"`
	// Token           string `toml:"token" comment:"optional, Bitbucket Cloud Token associated to username, used to add comment on Pull Request" json:"-"`
}

func (s BitbucketCloudConfiguration) check() error {
	if s.ProxyWebhook != "" && !strings.Contains(s.ProxyWebhook, "://") {
		return fmt.Errorf("Bitbucket proxy webhook must have the HTTP scheme")
	}
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
	return nil
}

func (s ServerConfiguration) check() error {
	if s.URL == "" {
		return fmt.Errorf("Invalid VCS server URL")
	}

	if s.Bitbucket != nil && s.BitbucketCloud != nil && s.Github != nil && s.Gitlab != nil {
		return fmt.Errorf("Invalid VCS server configuration")
	}

	if s.Bitbucket != nil {
		if err := s.Bitbucket.check(); err != nil {
			return err
		}
	}

	if s.BitbucketCloud != nil {
		if err := s.BitbucketCloud.check(); err != nil {
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

// GerritServerConfiguration represents the gerrit configuration
type GerritServerConfiguration struct {
	Status struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server" json:"disable"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server" json:"show_detail"`
	}
	DisableGerritEvent bool `toml:"disableGerritEvent" comment:"Does gerrit event stream are supported by VCS Server" json:"disable_gerrit_event"`
	SSHPort            int  `toml:"sshport" default:"29418" commented:"true" comment:"SSH port of gerrit"`
	EventStream        struct {
		User       string `toml:"user" default:"myuser" commented:"true" comment:"User to access to gerrit event stream"`
		PrivateKey string `toml:"privateKey" default:"" commented:"true" comment:"Private key of the user who access to gerrit event stream"`
	}
	Reviewer struct {
		User  string `toml:"user" default:"myreviewer" commented:"true" comment:"User that review changes"`
		Token string `toml:"token" default:"" commented:"true" comment:"Token of the reviewer"`
	}
}
