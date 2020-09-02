package repositories

// Service is the stuct representing a vcs µService
import (
	"path/filepath"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Service is the repostories service
type Service struct {
	service.Common
	Cfg    Configuration
	Router *api.Router
	Cache  cache.Store
	dao    dao
}

// Configuration is the vcs configuration structure
type Configuration struct {
	Name                  string `toml:"name" comment:"Name of this CDS Repositories Service\n Enter a name to enable this service" json:"name"`
	Basedir               string `toml:"basedir" comment:"Root directory where the service will store all checked-out repositories" json:"basedir"`
	OperationRetention    int    `toml:"operationRetention" comment:"Operation retention in redis store (in days)" default:"5" json:"operationRetention"`
	RepositoriesRetention int    `toml:"repositoriesRetention" comment:"Re retention on the filesystem (in days)" default:"10" json:"repositoriesRetention"`
	HTTP                  struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8085" json:"port"`
	} `toml:"http" comment:"######################\n CDS Repositories HTTP Configuration \n######################" json:"http"`
	URL   string                          `default:"http://localhost:8085" json:"url"`
	API   service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Cache struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS Repositories Cache Settings \n######################" json:"cache"`
}

// Repo retiens a sdk.OperationRepo from an sdk.Operation
func (s Service) Repo(op sdk.Operation) *sdk.OperationRepo {
	r := new(sdk.OperationRepo)
	r.URL = op.URL
	r.Basedir = filepath.Join(s.Cfg.Basedir, r.ID())
	r.RepositoryStrategy = op.RepositoryStrategy
	return r
}
