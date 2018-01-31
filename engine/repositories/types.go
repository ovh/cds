package repositories

// Service is the stuct representing a vcs µService
import (
	"encoding/base64"
	"path/filepath"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// Service is the repostories service
type Service struct {
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	cds    cdsclient.Interface
	hash   string
	dao    dao
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name    string `toml:"name" comment:"Name of this CDS Repositories Service"`
	Basedir string `toml:"basedir" comment:"Root directory where the service will store all checked-out repositories"`
	HTTP    struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8086" toml:"name"`
	} `toml:"http" comment:"######################\n CDS Repositories HTTP Configuration \n######################"`
	URL string `default:"http://localhost:8084"`
	API struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081"`
			Insecure bool   `toml:"insecure" commented:"true"`
		} `toml:"http"`
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
	} `toml:"cache" comment:"######################\n CDS Repositories Cache Settings \n######################"`
}

// Operation is the main business object use in repositories service
type Operation struct {
	UUID               string                 `json:"uuid"`
	URL                string                 `json:"url"`
	RepositoryStrategy sdk.RepositoryStrategy `json:"strategy,omitempty"`
	Setup              struct {
		Checkout OperationCheckout `json:"checkout,omitempty"`
	} `json:"setup,omitempty"`
	LoadFiles OperationLoadFiles `json:"load_files,omitempty"`
	Status    OperationStatus    `json:"status,omitempty"`
	Error     string             `json:"error,omitempty"`
}

func (s Service) Repo(op Operation) *Repo {
	r := new(Repo)
	r.URL = op.URL
	r.Basedir = filepath.Join(s.Cfg.Basedir, r.ID())
	r.RepositoryStrategy = op.RepositoryStrategy
	return r
}

type OperationLoadFiles struct {
	Pattern string            `json:"pattern,omitempty"`
	Results map[string][]byte `json:"results,omitempty"`
}

type OperationCheckout struct {
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
}

type OperationStatus int

const (
	OperationStatusPending OperationStatus = iota
	OperationStatusProcessing
	OperationStatusDone
	OperationStatusError
)

type Repo struct {
	Basedir            string
	URL                string
	RepositoryStrategy sdk.RepositoryStrategy
}

func (r Repo) ID() string {
	return base64.StdEncoding.EncodeToString([]byte(r.URL))
}
