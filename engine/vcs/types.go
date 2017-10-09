package vcs

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Service is the stuct representing a vcs µService
type Service struct {
	Cfg     Configuration
	Router  *api.Router
	Cache   cache.Store
	cds     cdsclient.Interface
	hash    string
	servers []ServerConfiguration
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS VCS Service"`
	HTTP struct {
		Port int `toml:"port" default:"8084" toml:"name"`
	} `toml:"http" comment:"######################\n CDS VCS HTTP Configuration \n######################"`
	URL string `default:"http://localhost:8084"`
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
	} `toml:"api" comment:"######################\n CDS API Settings \n######################`
	Cache struct {
		TTL   int `toml:"ttl" default:"60"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379"`
			Password string `toml:"password"`
		} `toml:"redis"`
	} `toml:"cache" comment:"######################\n CDS VCS Cache Settings \n######################"`
	Servers []ServerConfiguration `toml:"servers" comment:"######################\n CDS VCS Server Settings \n######################"`
}

// ServerConfiguration is the configuration for a VCS server
type ServerConfiguration struct {
	UUID      string                        `toml:"-" json:"uuid"`
	Name      string                        `toml:"name" comment:"Name of this VCS Server" json:"name"`
	URL       string                        `toml:"url" comment:"URL of this VCS Server" json:"url"`
	Github    *GithubServerConfiguration    `toml:"github" json:"github,omitempty"`
	Gitlab    *GitlabServerConfiguration    `toml:"gitlab" json:"gitlab,omitempty"`
	Bitbucket *BitbucketServerConfiguration `toml:"bitbucket" json:"bitbucket,omitempty"`
}

// GithubServerConfiguration represents the github configuration
type GithubServerConfiguration struct {
	Secret string `toml:"secret"`
	Status struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server"`
	}
	DisableWebHooks         bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server"`
	DisableWebHooksCreation bool `toml:"disableWebHooksCreation" comment:"Does webhooks creation are supported by VCS Server"`
	DisablePolling          bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server"`
}

func (s GithubServerConfiguration) check() error {
	if s.Secret == "" {
		return errGithubConfigurationError
	}
	return nil
}

var errGithubConfigurationError = fmt.Errorf("Github configuration Error")

// GitlabServerConfiguration represents the gitlab configuration
type GitlabServerConfiguration struct {
	Secret string `toml:"secret"`
	Status struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server"`
	}
	DisableWebHooks         bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server"`
	DisableWebHooksCreation bool `toml:"disableWebHooksCreation" comment:"Does webhooks creation are supported by VCS Server"`
	DisablePolling          bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server"`
}

func (s GitlabServerConfiguration) check() error {
	return nil
}

// BitbucketServerConfiguration represents the bitbucket configuration
type BitbucketServerConfiguration struct {
	ConsumerKey string `toml:"consumerKey"`
	PrivateKey  string `toml:"privateKey"`
	Status      struct {
		Disable    bool `toml:"disable" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push statuses on the VCS server"`
		ShowDetail bool `toml:"showDetail" default:"false" commented:"true" comment:"Set to true if you don't want CDS to push CDS URL in statuses on the VCS server"`
	}
	DisableWebHooks         bool `toml:"disableWebHooks" comment:"Does webhooks are supported by VCS Server"`
	DisableWebHooksCreation bool `toml:"disableWebHooksCreation" comment:"Does webhooks creation are supported by VCS Server"`
	DisablePolling          bool `toml:"disablePolling" comment:"Does polling is supported by VCS Server"`
}

func (s BitbucketServerConfiguration) check() error {
	return nil
}

func (s *Service) addServerConfiguration(c *ServerConfiguration) error {
	if c.UUID == "" {
		uuid, erruuid := sessionstore.NewSessionKey()
		if erruuid != nil {
			return sdk.WrapError(erruuid, "Unable to generate UUID")
		}
		c.UUID = string(uuid)
	}

	if err := c.check(); err != nil {
		return sdk.WrapError(err, "Unable to add server configuration")
	}
	s.servers = append(s.servers, *c)
	return nil
}

func (s ServerConfiguration) check() error {
	if s.UUID == "" {
		return fmt.Errorf("Invalid VCS server uuid")
	}

	if s.Name == "" {
		return fmt.Errorf("Invalid VCS server name")
	}

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

type Server interface {
	AuthorizeRedirect() (string, string, error)
	AuthorizeToken(string, string) (string, string, error)
	GetAuthorized(string, string) (AuthorizedClient, error)
}

type AuthorizedClient interface {
	//Repos
	Repos() ([]sdk.VCSRepo, error)
	RepoByFullname(fullname string) (sdk.VCSRepo, error)

	//Branches
	Branches(string) ([]sdk.VCSBranch, error)
	Branch(string, string) (*sdk.VCSBranch, error)

	//Commits
	Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error)
	Commit(repo, hash string) (sdk.VCSCommit, error)

	// PullRequests
	PullRequests(string) ([]sdk.VCSPullRequest, error)

	//Hooks
	CreateHook(repo, url string) error
	DeleteHook(repo, url string) error

	//Events
	GetEvents(repo string, dateRef time.Time) ([]interface{}, time.Duration, error)
	PushEvents(string, []interface{}) ([]sdk.VCSPushEvent, error)
	CreateEvents(string, []interface{}) ([]sdk.VCSCreateEvent, error)
	DeleteEvents(string, []interface{}) ([]sdk.VCSDeleteEvent, error)
	PullRequestEvents(string, []interface{}) ([]sdk.VCSPullRequestEvent, error)

	// Set build status on repository
	SetStatus(event sdk.Event) error

	// Release
	Release(repo, tagName, releaseTitle, releaseDescription string) (*sdk.VCSRelease, error)
	UploadReleaseFile(repo string, release *sdk.VCSRelease, runArtifact sdk.WorkflowNodeRunArtifact, file *bytes.Buffer) error
}
