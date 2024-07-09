package hooks

import (
	"crypto/rsa"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Task execution status
const (
	TaskExecutionEnqueued  = "ENQUEUED"
	TaskExecutionDoing     = "DOING"
	TaskExecutionDone      = "DONE"
	TaskExecutionScheduled = "SCHEDULED"
)

const (
	repositoryEventRootKey        = "hooks:events:repository"
	repositoryEventLockRootKey    = "hooks:events:lock"
	repositoryLock                = "hooks:lock:repository"
	repositoryRootKey             = "hooks:repository"
	repositoryEventQueue          = "hooks:queue:repository:event"
	repositoryEventCallbackQueue  = "hooks:queue:repository:event:callback"
	repositoryEventInProgressKey  = "hooks:queue:repository:event:inprogress"
	schedulerNextExecutionRootKey = "hooks:queue:schedulers"
	scheduleDefinitionRootKey     = "hooks:v2:definition:schedulers"
	schedulerExecutionLockRootKey = "hooks:v2:executions:lock"
)

// Service is the stuct representing a hooks µService
type Service struct {
	service.Common
	Cfg                     Configuration
	Router                  *api.Router
	Cache                   cache.Store
	Dao                     dao
	Maintenance             bool
	WebHooksParsedPublicKey *rsa.PublicKey
}

// Configuration is the hooks configuration structure
type Configuration struct {
	Name                        string                          `toml:"name" comment:"Name of this CDS Hooks Service\n Enter a name to enable this service" json:"name"`
	HTTP                        service.HTTPRouterConfiguration `toml:"http" comment:"######################\n CDS Hooks HTTP Configuration \n######################" json:"http"`
	URL                         string                          `toml:"url" default:"http://localhost:8083" json:"url"`
	URLPublic                   string                          `toml:"urlPublic" default:"http://localhost:8080/cdshooks" comment:"Public url for external call (webhook)" json:"urlPublic"`
	OldRepositoryEventRetry     int64                           `toml:"oldRepositoryEventRetry" default:"1" comment:"Delay of old repository event check in minute" json:"oldRepositoryEventRetry"`
	OldRepositoryEventQueueLen  int                             `toml:"oldRepositoryEventQueueLen" default:"200" comment:"Maximum queue len allowed to re-enqueue old repository events" json:"oldRepositoryEventQueueLen"`
	DisableRepositoryEventRetry bool                            `toml:"disableRepositoryEventRetry" default:"false" comment:"" json:"disableRepositoryEventRetry"`
	RetryDelay                  int64                           `toml:"retryDelay" default:"120" comment:"Execution retry delay in seconds" json:"retryDelay"`
	RetryError                  int64                           `toml:"retryError" default:"3" comment:"Retry execution while this number of error is not reached" json:"retryError"`
	ExecutionHistory            int                             `toml:"executionHistory" default:"10" comment:"Number of execution to keep" json:"executionHistory"`
	RepositoryEventRetention    int                             `toml:"repositoryEventRetention" default:"30" comment:"Number of repository event to keep" json:"repositoryEventRetention"`
	Disable                     bool                            `toml:"disable" default:"false" comment:"Disable all hooks executions" json:"disable"`
	API                         service.APIServiceConfiguration `toml:"api" comment:"######################\n CDS API Settings \n######################" json:"api"`
	Cache                       struct {
		TTL   int           `toml:"ttl" default:"60" json:"ttl"`
		Redis sdk.RedisConf `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS Hooks Cache Settings \n######################" json:"cache"`
	WebhooksPublicKeySign string `toml:"webhooksPublicKeySign" comment:"Public key to check call signature on handler /v2/webhook/repository"`
	RepositoryWebHookKey  string `toml:"repositoryWebHookKey" comment:"Secret key used to generate repository webhook secret"`
}
