package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/version"
	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Configuration is the configuraton structure for CDS API
type Configuration struct {
	URL struct {
		API string `toml:"api" default:"http://localhost:8081" json:"api"`
		UI  string `toml:"ui" default:"http://localhost:2015" json:"ui"`
	} `toml:"url" comment:"#####################\n CDS URLs Settings \n####################" json:"url"`
	HTTP struct {
		Addr       string `toml:"addr" default:"" commented:"true" comment:"Listen HTTP address without port, example: 127.0.0.1" json:"addr"`
		Port       int    `toml:"port" default:"8081" json:"port"`
		SessionTTL int    `toml:"sessionTTL" default:"60" json:"sessionTTL"`
	} `toml:"http" json:"http"`
	GRPC struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen GRPC address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8082" json:"port"`
	} `toml:"grpc" json:"grpc"`
	Secrets struct {
		Key string `toml:"key" json:"-"`
	} `toml:"secrets" json:"secrets"`
	Database database.DBConfiguration `toml:"database" comment:"################################\n Postgresql Database settings \n###############################" json:"database"`
	Cache    struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS Cache Settings \n#####################\n" json:"cache"`
	Directories struct {
		Download string `toml:"download" default:"/tmp/cds/download" json:"download"`
		Keys     string `toml:"keys" default:"/tmp/cds/keys" json:"keys"`
	} `toml:"directories" json:"directories"`
	Auth struct {
		DefaultGroup     string `toml:"defaultGroup" default:"" comment:"The default group is the group in which every new user will be granted at signup" json:"defaultGroup"`
		SharedInfraToken string `toml:"sharedInfraToken" default:"" comment:"Token for shared.infra group. This value will be used when shared.infra will be created\nat first CDS launch. This token can be used by CDS CLI, Hatchery, etc...\nThis is mandatory." json:"-"`
		RSAPrivateKey    string `toml:"rsaPrivateKey" default:"" comment:"The RSA Private Key used to sign and verify the JWT Tokens issued by the API \nThis is mandatory." json:"-"`
		LDAP             struct {
			Enable   bool   `toml:"enable" default:"false" json:"enable"`
			Host     string `toml:"host" json:"host"`
			Port     int    `toml:"port" default:"636" json:"port"`
			SSL      bool   `toml:"ssl" default:"true" json:"ssl"`
			Base     string `toml:"base" default:"dc=myorganization,dc=com" json:"base"`
			DN       string `toml:"dn" default:"uid=%s,ou=people,dc=myorganization,dc=com" json:"dn"`
			Fullname string `toml:"fullname" default:"{{.givenName}} {{.sn}}" json:"fullname"`
			BindDN   string `toml:"bindDN" default:"" comment:"Define it if ldapsearch need to be authenticated" json:"bindDN"`
			BindPwd  string `toml:"bindPwd" default:"" comment:"Define it if ldapsearch need to be authenticated" json:"-"`
		} `toml:"ldap" json:"ldap"`
		Local struct {
			SignupAllowedDomains string `toml:"signupAllowedDomains" default:"" comment:"Allow signup from selected domains only - comma separated. Example: your-domain.com,another-domain.com" commented:"true" json:"signupAllowedDomains"`
		} `toml:"local" json:"local"`
	} `toml:"auth" comment:"##############################\n CDS Authentication Settings#\n#############################" json:"auth"`
	SMTP struct {
		Disable  bool   `toml:"disable" default:"true" json:"disable"`
		Host     string `toml:"host" json:"host"`
		Port     string `toml:"port" json:"port"`
		TLS      bool   `toml:"tls" json:"tls"`
		User     string `toml:"user" json:"user"`
		Password string `toml:"password" json:"-"`
		From     string `toml:"from" default:"no-reply@cds.local" json:"from"`
	} `toml:"smtp" comment:"#####################\n# CDS SMTP Settings \n####################" json:"smtp"`
	Artifact struct {
		Mode  string `toml:"mode" default:"local" comment:"swift, awss3 or local" json:"mode"`
		Local struct {
			BaseDirectory string `toml:"baseDirectory" default:"/tmp/cds/artifacts" json:"baseDirectory"`
		} `toml:"local"`
		Openstack struct {
			URL             string `toml:"url" comment:"Authentication Endpoint, generally value of $OS_AUTH_URL" json:"url"`
			Username        string `toml:"username" comment:"Openstack Username, generally value of $OS_USERNAME" json:"username"`
			Password        string `toml:"password" comment:"Openstack Password, generally value of $OS_PASSWORD" json:"-"`
			Tenant          string `toml:"tenant" comment:"Openstack Tenant, generally value of $OS_TENANT_NAME, v2 auth only" json:"tenant"`
			Domain          string `toml:"domain" comment:"Openstack Domain, generally value of $OS_DOMAIN_NAME, v3 auth only" json:"domain"`
			Region          string `toml:"region" comment:"Region, generally value of $OS_REGION_NAME" json:"region"`
			ContainerPrefix string `toml:"containerPrefix" comment:"Use if your want to prefix containers for CDS Artifacts" json:"containerPrefix"`
			DisableTempURL  bool   `toml:"disableTempURL" default:"false" commented:"true" comment:"True if you want to disable Temporary URL in file upload" json:"disableTempURL"`
		} `toml:"openstack" json:"openstack"`
		AWSS3 struct {
			BucketName          string `toml:"bucketName" json:"bucketName" comment:"Name of the S3 bucket to use when storing artifacts"`
			Region              string `toml:"region" json:"region" default:"us-east-1" comment:"The AWS region"`
			Prefix              string `toml:"prefix" json:"prefix" comment:"A subfolder of the bucket to store objects in, if left empty will store at the root of the bucket"`
			AuthFromEnvironment bool   `toml:"authFromEnv" json:"authFromEnv" default:"false" comment:"Pull S3 auth information from env vars AWS_SECRET_ACCESS_KEY and AWS_SECRET_KEY_ID"`
			SharedCredsFile     string `toml:"sharedCredsFile" json:"sharedCredsFile" comment:"The path for the AWS credential file, used with profile"`
			Profile             string `toml:"profile" json:"profile" comment:"The profile within the AWS credentials file to use"`
			AccessKeyID         string `toml:"accessKeyId" json:"accessKeyId" comment:"A static AWS Secret Key ID"`
			SecretAccessKey     string `toml:"secretAccessKey" json:"-" comment:"A static AWS Secret Access Key"`
			SessionToken        string `toml:"sessionToken" json:"-" comment:"A static AWS session token"`
		} `toml:"awss3" json:"awss3"`
	} `toml:"artifact" comment:"Either filesystem local storage or Openstack Swift Storage are supported" json:"artifact"`
	Events struct {
		Kafka struct {
			Enabled         bool   `toml:"enabled" json:"enabled"`
			Broker          string `toml:"broker" json:"broker"`
			Topic           string `toml:"topic" json:"topic"`
			User            string `toml:"user" json:"user"`
			Password        string `toml:"password" json:"-"`
			MaxMessageBytes int    `toml:"maxmessagebytes" default:"10000000" json:"maxmessagebytes"`
		} `toml:"kafka" json:"kafka"`
	} `toml:"events" comment:"#######################\n CDS Events Settings \n######################" json:"events"`
	Features struct {
		Izanami struct {
			APIURL       string `toml:"apiurl" json:"apiurl"`
			ClientID     string `toml:"clientid" json:"-"`
			ClientSecret string `toml:"clientsecret" json:"-"`
			Token        string `toml:"token" comment:"Token shared between Izanami and CDS to be able to send webhooks from izanami" json:"-"`
		} `toml:"izanami" comment:"Feature flipping provider: https://maif.github.io/izanami" json:"izanami"`
	} `toml:"features" comment:"###########################\n CDS Features flipping Settings \n##########################" json:"features"`
	Vault struct {
		ConfigurationKey string `toml:"configurationKey" json:"-"`
	} `toml:"vault" json:"vault"`
	Providers []ProviderConfiguration `toml:"providers" comment:"###########################\n CDS Providers Settings \n##########################" json:"providers"`
	Services  []ServiceConfiguration  `toml:"services" comment:"###########################\n CDS Services Settings \n##########################" json:"services"`
	Status    struct {
		API struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of API is running, an alert will on Global/API be created on /mon/status" json:"minInstance"`
		} `toml:"api" json:"api"`
		DBMigrate struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of dbmigrate service is running, an alert on Global/dbmigrate will be created on /mon/status" json:"minInstance"`
		} `toml:"dbmigrate" json:"dbmigrate"`
		ElasticSearch struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of elasticsearch service is running, an alert on Global/elasticsearch will be created on /mon/status" json:"minIntance"`
		} `toml:"elasticsearch" json:"elasticsearch"`
		Hatchery struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of hatchery service is running, an alert on Global/hatchery will be created on /mon/status" json:"minInstance"`
		} `toml:"hatchery" json:"hatchery"`
		Hooks struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of hooks service is running, an alert on Global/hooks will be created on /mon/status" json:"minInstance"`
		} `toml:"hooks" json:"hooks"`
		Repositories struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of repositories service is running, an alert on Global/hooks will be created on /mon/status" json:"minInstance"`
		} `toml:"repositories" json:"repositories"`
		VCS struct {
			MinInstance int `toml:"minInstance" default:"1" comment:"if less than minInstance of vcs service is running, an alert will on Global/vcs be created on /mon/status" json:"minInstance"`
		} `toml:"vcs" json:"vcs"`
	} `toml:"status" comment:"###########################\n CDS Status Settings \n Documentation: https://ovh.github.io/cds/hosting/monitoring/ \n##########################" json:"status"`
	DefaultOS   string `toml:"defaultOS" default:"linux" comment:"if no model and os/arch is specified in your job's requirements then spawn worker on this operating system (example: freebsd, linux, windows)" json:"defaultOS"`
	DefaultArch string `toml:"defaultArch" default:"amd64" comment:"if no model and no os/arch is specified in your job's requirements then spawn worker on this architecture (example: amd64, arm, 386)" json:"defaultArch"`
	Graylog     struct {
		AccessToken string `toml:"accessToken" json:"-"`
		Stream      string `toml:"stream" json:"-"`
		URL         string `toml:"url" comment:"Example: http://localhost:9000" json:"url"`
	} `toml:"graylog" json:"graylog" comment:"###########################\n Graylog Search. \n When CDS API generates errors, you can fetch them with cdsctl. \n Examples: \n $ cdsctl admin errors get <error-id> \n $ cdsctl admin errors get 55f6e977-d39b-11e8-8513-0242ac110007 \n##########################"`
	Log struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
}

// ProviderConfiguration is the piece of configuration for each provider authentication
type ProviderConfiguration struct {
	Name  string `toml:"name" json:"name"`
	Token string `toml:"token" json:"-"`
}

// ServiceConfiguration is the configuration of external service
type ServiceConfiguration struct {
	Name       string `toml:"name" json:"name"`
	URL        string `toml:"url" json:"url"`
	Port       string `toml:"port" json:"port"`
	Path       string `toml:"path" json:"path"`
	HealthURL  string `toml:"healthUrl" json:"healthUrl"`
	HealthPort string `toml:"healthPort" json:"healthPort"`
	HealthPath string `toml:"healthPath" json:"healthPath"`
	Type       string `toml:"type" json:"type"`
}

// DefaultValues is the struc for API Default configuration default values
type DefaultValues struct {
	ServerSecretsKey     string
	AuthSharedInfraToken string
	// For LDAP Client
	LDAPBase  string
	GivenName string
	SN        string
	BindDN    string
	BindPwd   string
}

const permProjectKey = "permProjectKey"

// New instanciates a new API object
func New() *API {
	return &API{}
}

// Service returns an instance of sdk.Service for the API
func (*API) Service() sdk.Service {
	return sdk.Service{
		LastHeartbeat: time.Time{},
		Type:          services.TypeAPI,
	}
}

// API is a struct containing the configuration, the router, the database connection factory and so on
type API struct {
	service.Common
	Router              *Router
	Config              Configuration
	DBConnectionFactory *database.DBConnectionFactory
	SharedStorage       objectstore.Driver
	StartupTime         time.Time
	Maintenance         bool
	eventsBroker        *eventsBroker
	warnChan            chan sdk.Event
	Cache               cache.Store
	Metrics             struct {
		WorkflowRunFailed        *stats.Int64Measure
		WorkflowRunStarted       *stats.Int64Measure
		Sessions                 *stats.Int64Measure
		nbUsers                  *stats.Int64Measure
		nbApplications           *stats.Int64Measure
		nbProjects               *stats.Int64Measure
		nbGroups                 *stats.Int64Measure
		nbPipelines              *stats.Int64Measure
		nbWorkflows              *stats.Int64Measure
		nbArtifacts              *stats.Int64Measure
		nbWorkerModels           *stats.Int64Measure
		nbWorkflowRuns           *stats.Int64Measure
		nbWorkflowNodeRuns       *stats.Int64Measure
		nbMaxWorkersBuilding     *stats.Int64Measure
		queue                    *stats.Int64Measure
		WorkflowRunsMarkToDelete *stats.Int64Measure
		WorkflowRunsDeleted      *stats.Int64Measure
	}
}

// ApplyConfiguration apply an object of type api.Configuration after checking it
func (a *API) ApplyConfiguration(config interface{}) error {
	if err := a.CheckConfiguration(config); err != nil {
		return err
	}

	var ok bool
	a.Config, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	a.Type = services.TypeAPI
	a.ServiceName = "cds-api"

	return nil
}

// DirectoryExists checks if the directory exists
func DirectoryExists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return s.IsDir(), err
}

// CheckConfiguration checks the validity of the configuration object
func (a *API) CheckConfiguration(config interface{}) error {
	aConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid API configuration")
	}

	if aConfig.URL.API == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	if aConfig.Directories.Download == "" {
		return fmt.Errorf("Invalid download directory (empty)")
	}

	if ok, err := DirectoryExists(aConfig.Directories.Download); !ok {
		if err := os.MkdirAll(aConfig.Directories.Download, os.FileMode(0700)); err != nil {
			return fmt.Errorf("Unable to create directory %s: %v", aConfig.Directories.Download, err)
		}
		log.Info("Directory %s has been created", aConfig.Directories.Download)
	} else if err != nil {
		return fmt.Errorf("Invalid download directory %s: %v", aConfig.Directories.Download, err)
	}

	if aConfig.Directories.Keys == "" {
		return fmt.Errorf("Invalid keys directory")
	}

	if ok, err := DirectoryExists(aConfig.Directories.Keys); !ok {
		if err := os.MkdirAll(aConfig.Directories.Keys, os.FileMode(0700)); err != nil {
			return fmt.Errorf("Unable to create directory %s: %v", aConfig.Directories.Keys, err)
		}
		log.Info("Directory %s has been created", aConfig.Directories.Keys)
	} else if err != nil {
		return fmt.Errorf("Invalid keys directory: %v", err)
	}

	switch aConfig.Artifact.Mode {
	case "local", "awss3", "openstack", "swift":
	default:
		return fmt.Errorf("Invalid artifact mode")
	}

	if aConfig.Artifact.Mode == "local" {
		if aConfig.Artifact.Local.BaseDirectory == "" {
			return fmt.Errorf("Invalid artifact local base directory (empty name)")
		}
		if ok, err := DirectoryExists(aConfig.Artifact.Local.BaseDirectory); !ok {
			if err := os.MkdirAll(aConfig.Artifact.Local.BaseDirectory, os.FileMode(0700)); err != nil {
				return fmt.Errorf("Unable to create directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
			}
			log.Info("Directory %s has been created", aConfig.Artifact.Local.BaseDirectory)
		} else if err != nil {
			return fmt.Errorf("Invalid artifact local base directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
		}
	}

	if len(aConfig.Secrets.Key) != 32 {
		return fmt.Errorf("Invalid secret key. It should be 32 bits (%d)", len(aConfig.Secrets.Key))
	}

	if aConfig.DefaultArch == "" {
		log.Warning(`You should add a default architecture in your configuration (example: defaultArch: "amd64"). It means if there is no model and os/arch requirement on your job then spawn on a worker based on this architecture`)
	}
	if aConfig.DefaultOS == "" {
		log.Warning(`You should add a default operating system in your configuration (example: defaultOS: "linux"). It means if there is no model and os/arch requirement on your job then spawn on a worker based on this OS`)
	}

	if (aConfig.DefaultOS == "" && aConfig.DefaultArch != "") || (aConfig.DefaultOS != "" && aConfig.DefaultArch == "") {
		return fmt.Errorf("You can't specify just defaultArch without defaultOS in your configuration and vice versa")
	}

	return nil
}

func getUserSession(c context.Context) string {
	i := c.Value(auth.ContextUserSession)
	if i == nil {
		return ""
	}
	u, ok := i.(string)
	if !ok {
		return ""
	}
	return u
}

func getGrantedUser(c context.Context) *sdk.GrantedUser {
	i := c.Value(ContextGrantedUser)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.GrantedUser)
	if !ok {
		return nil
	}
	return u
}

func deprecatedGetUser(c context.Context) *sdk.User {
	i := c.Value(auth.ContextUser)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.User)
	if !ok {
		return nil
	}
	return u
}

func getProvider(c context.Context) *string {
	i := c.Value(auth.ContextProvider)
	if i == nil {
		return nil
	}
	u, ok := i.(string)
	if !ok {
		return nil
	}
	return &u
}

func getAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func isServiceOrWorker(r *http.Request) bool {
	switch getAgent(r) {
	case sdk.ServiceAgent:
		return true
	case sdk.WorkerAgent:
		return true
	default:
		return false
	}
}

func getWorker(c context.Context) *sdk.Worker {
	i := c.Value(auth.ContextWorker)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Worker)
	if !ok {
		return nil
	}
	return u
}

func getHatchery(c context.Context) *sdk.Service {
	i := c.Value(auth.ContextHatchery)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Service)
	if !ok {
		return nil
	}
	return u
}

func getService(c context.Context) *sdk.Service {
	i := c.Value(auth.ContextService)
	if i == nil {
		return nil
	}
	u, ok := i.(*sdk.Service)
	if !ok {
		return nil
	}
	return u
}

func (a *API) mustDB() *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}
	return db
}

func (a *API) mustDBWithCtx(ctx context.Context) *gorp.DbMap {
	db := a.DBConnectionFactory.GetDBMap()
	db = db.WithContext(ctx).(*gorp.DbMap)
	if db == nil {
		panic(fmt.Errorf("Database unavailable"))
	}

	return db
}

// Serve will start the http api server
func (a *API) Serve(ctx context.Context) error {
	log.Info("Starting CDS API Server %s", sdk.VERSION)

	a.StartupTime = time.Now()

	// Checking downloadable binaries
	resources := sdk.AllDownloadableResourcesWithAvailability(a.Config.Directories.Download)
	var hasWorker, hasCtl, hasEngine bool
	for _, r := range resources {
		if r.Available != nil && *r.Available {
			switch r.Name {
			case "worker":
				hasWorker = true
			case "cdsctl":
				hasCtl = true
			case "engine":
				hasEngine = true
			}
		}
	}
	if !hasEngine {
		log.Error("engine is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", a.Config.Directories.Download)
	}
	if !hasCtl {
		log.Error("cdsctl is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", a.Config.Directories.Download)
	}
	if !hasWorker {
		// If no worker, let's exit because CDS for run anything
		log.Error("worker is unavailable for download. Please check your configuration file or the %s directory", a.Config.Directories.Download)
		return errors.New("worker binary unavailabe")
	}

	//Initialize secret driver
	secret.Init(a.Config.Secrets.Key)

	//Initialize the jwt layer
	if a.Config.Auth.RSAPrivateKey != "" { // Temporary condition...
		if err := accesstoken.Init(a.Name, []byte(a.Config.Auth.RSAPrivateKey)); err != nil {
			return fmt.Errorf("unable to initialize the JWT Layer: %v", err)
		}
	}

	//Initialize mail package
	log.Info("Initializing mail driver...")
	mail.Init(a.Config.SMTP.User,
		a.Config.SMTP.Password,
		a.Config.SMTP.From,
		a.Config.SMTP.Host,
		a.Config.SMTP.Port,
		a.Config.SMTP.TLS,
		a.Config.SMTP.Disable)

	// Initialize feature packages
	log.Info("Initializing feature flipping with izanami %s", a.Config.Features.Izanami.APIURL)
	if a.Config.Features.Izanami.APIURL != "" {
		if err := feature.Init(a.Config.Features.Izanami.APIURL, a.Config.Features.Izanami.ClientID, a.Config.Features.Izanami.ClientSecret); err != nil {
			return fmt.Errorf("Feature flipping not enabled with izanami: %v", err)
		}
	}

	//Initialize artifacts storage
	log.Info("Initializing %s objectstore...", a.Config.Artifact.Mode)
	var objectstoreKind objectstore.Kind
	switch a.Config.Artifact.Mode {
	case "openstack":
		objectstoreKind = objectstore.Openstack
	case "swift":
		objectstoreKind = objectstore.Swift
	case "awss3":
		objectstoreKind = objectstore.AWSS3
	case "filesystem", "local":
		objectstoreKind = objectstore.Filesystem
	default:
		return fmt.Errorf("unsupported objecstore mode : %s", a.Config.Artifact.Mode)
	}

	cfg := objectstore.Config{
		Kind: objectstoreKind,
		Options: objectstore.ConfigOptions{
			AWSS3: objectstore.ConfigOptionsAWSS3{
				Prefix:              a.Config.Artifact.AWSS3.Prefix,
				SecretAccessKey:     a.Config.Artifact.AWSS3.SecretAccessKey,
				AccessKeyID:         a.Config.Artifact.AWSS3.AccessKeyID,
				Profile:             a.Config.Artifact.AWSS3.Profile,
				SharedCredsFile:     a.Config.Artifact.AWSS3.SharedCredsFile,
				AuthFromEnvironment: a.Config.Artifact.AWSS3.AuthFromEnvironment,
				BucketName:          a.Config.Artifact.AWSS3.BucketName,
				Region:              a.Config.Artifact.AWSS3.Region,
				SessionToken:        a.Config.Artifact.AWSS3.SessionToken,
			},
			Openstack: objectstore.ConfigOptionsOpenstack{
				Address:         a.Config.Artifact.Openstack.URL,
				Username:        a.Config.Artifact.Openstack.Username,
				Password:        a.Config.Artifact.Openstack.Password,
				Tenant:          a.Config.Artifact.Openstack.Tenant,
				Domain:          a.Config.Artifact.Openstack.Domain,
				Region:          a.Config.Artifact.Openstack.Region,
				ContainerPrefix: a.Config.Artifact.Openstack.ContainerPrefix,
				DisableTempURL:  a.Config.Artifact.Openstack.DisableTempURL,
			},
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: a.Config.Artifact.Local.BaseDirectory,
			},
		},
	}

	// DEPRECATED
	// API Storage will be a public integration
	var errStorage error
	a.SharedStorage, errStorage = objectstore.Init(ctx, cfg)
	if errStorage != nil {
		return fmt.Errorf("cannot initialize storage: %v", errStorage)
	}

	log.Info("Initializing database connection...")
	//Intialize database
	var errDB error
	a.DBConnectionFactory, errDB = database.Init(
		a.Config.Database.User,
		a.Config.Database.Role,
		a.Config.Database.Password,
		a.Config.Database.Name,
		a.Config.Database.Host,
		a.Config.Database.Port,
		a.Config.Database.SSLMode,
		a.Config.Database.ConnectTimeout,
		a.Config.Database.Timeout,
		a.Config.Database.MaxConn)
	if errDB != nil {
		return fmt.Errorf("cannot connect to database: %v", errDB)
	}

	log.Info("Bootstrapping database...")
	defaultValues := sdk.DefaultValues{
		DefaultGroupName: a.Config.Auth.DefaultGroup,
		SharedInfraToken: a.Config.Auth.SharedInfraToken,
	}
	if err := bootstrap.InitiliazeDB(defaultValues, a.DBConnectionFactory.GetDBMap); err != nil {
		return fmt.Errorf("cannot setup databases: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow hook models: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowOutgoingHookModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow outgoing hook models: %v", err)
	}

	if err := integration.CreateBuiltinModels(a.DBConnectionFactory.GetDBMap()); err != nil {
		return fmt.Errorf("cannot setup integrations: %v", err)
	}

	log.Info("Initializing redis cache on %s...", a.Config.Cache.Redis.Host)
	//Init the cache
	var errCache error
	a.Cache, errCache = cache.New(
		a.Config.Cache.Redis.Host,
		a.Config.Cache.Redis.Password,
		a.Config.Cache.TTL)
	if errCache != nil {
		return fmt.Errorf("cannot connect to cache store: %v", errCache)
	}

	log.Info("Initializing HTTP router")
	a.Router = &Router{
		Mux:        mux.NewRouter(),
		Background: ctx,
	}
	a.InitRouter()
	if err := a.Router.InitMetrics("cds-api", a.Name); err != nil {
		log.Error("unable to init router metrics: %v", err)
	}

	log.Info("Initializing Metrics")
	if err := a.initMetrics(ctx); err != nil {
		log.Error("unable to init api metrics: %v", err)
	}

	//Intialize notification package
	notification.Init(a.Config.URL.UI)

	log.Info("Initializing Authentication driver...")
	// Initialize the auth driver
	var authMode string
	var authOptions interface{}
	switch a.Config.Auth.LDAP.Enable {
	case true:
		authMode = "ldap"
		authOptions = auth.LDAPConfig{
			Host:         a.Config.Auth.LDAP.Host,
			Port:         a.Config.Auth.LDAP.Port,
			Base:         a.Config.Auth.LDAP.Base,
			DN:           a.Config.Auth.LDAP.DN,
			SSL:          a.Config.Auth.LDAP.SSL,
			UserFullname: a.Config.Auth.LDAP.Fullname,
			BindDN:       a.Config.Auth.LDAP.BindDN,
			BindPwd:      a.Config.Auth.LDAP.BindPwd,
		}
	default:
		authMode = "local"
	}

	storeOptions := sessionstore.Options{
		TTL:   a.Config.HTTP.SessionTTL * 60, // Second to minutes
		Cache: a.Cache,
	}

	var errdriver error
	a.Router.AuthDriver, errdriver = auth.GetDriver(ctx, authMode, authOptions, storeOptions, a.DBConnectionFactory.GetDBMap)
	if errdriver != nil {
		return fmt.Errorf("error: %v", errdriver)
	}

	log.Info("Initializing event broker...")
	kafkaOptions := event.KafkaConfig{
		Enabled:         a.Config.Events.Kafka.Enabled,
		BrokerAddresses: a.Config.Events.Kafka.Broker,
		User:            a.Config.Events.Kafka.User,
		Password:        a.Config.Events.Kafka.Password,
		Topic:           a.Config.Events.Kafka.Topic,
		MaxMessageByte:  a.Config.Events.Kafka.MaxMessageBytes,
	}
	if err := event.Initialize(kafkaOptions, a.Cache); err != nil {
		log.Error("error while initializing event system: %s", err)
	} else {
		go event.DequeueEvent(ctx)
	}

	a.warnChan = make(chan sdk.Event)
	event.Subscribe(a.warnChan)

	log.Info("Initializing internal routines...")
	sdk.GoRoutine(ctx, "maintenance.Subscribe", func(ctx context.Context) {
		a.listenMaintenance(ctx)
	}, a.PanicDump())

	sdk.GoRoutine(ctx, "worker.Initialize", func(ctx context.Context) {
		if err := worker.Initialize(ctx, a.DBConnectionFactory.GetDBMap, a.Cache); err != nil {
			log.Error("error while initializing workers routine: %s", err)
		}
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "action.ComputeAudit", func(ctx context.Context) {
		action.ComputeAudit(ctx, a.DBConnectionFactory.GetDBMap)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "pipeline.ComputeAudit", func(ctx context.Context) {
		pipeline.ComputeAudit(ctx, a.DBConnectionFactory.GetDBMap)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "workflow.ComputeAudit", func(ctx context.Context) {
		workflow.ComputeAudit(ctx, a.DBConnectionFactory.GetDBMap)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "workflowtemplate.ComputeAudit", func(ctx context.Context) {
		workflowtemplate.ComputeAudit(ctx, a.DBConnectionFactory.GetDBMap)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "warning.Start", func(ctx context.Context) {
		warning.Start(ctx, a.DBConnectionFactory.GetDBMap, a.warnChan)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "auditCleanerRoutine(ctx", func(ctx context.Context) {
		auditCleanerRoutine(ctx, a.DBConnectionFactory.GetDBMap)
	})
	sdk.GoRoutine(ctx, "repositoriesmanager.ReceiveEvents", func(ctx context.Context) {
		repositoriesmanager.ReceiveEvents(ctx, a.DBConnectionFactory.GetDBMap, a.Cache)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "services.KillDeadServices", func(ctx context.Context) {
		services.KillDeadServices(ctx, a.mustDB)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "broadcast.Initialize", func(ctx context.Context) {
		broadcast.Initialize(ctx, a.DBConnectionFactory.GetDBMap)
	}, a.PanicDump())
	sdk.GoRoutine(ctx, "api.serviceAPIHeartbeat", func(ctx context.Context) {
		a.serviceAPIHeartbeat(ctx)
	}, a.PanicDump())

	//Temporary migration code
	//DEPRECATED Migrations
	sdk.GoRoutine(ctx, "migrate.KeyMigration", func(ctx context.Context) {
		migrate.KeyMigration(a.Cache, a.DBConnectionFactory.GetDBMap, &sdk.User{Admin: true})
	}, a.PanicDump())

	migrate.Add(sdk.Migration{Name: "Permissions", Release: "0.37.3", Mandatory: true, ExecFunc: func(ctx context.Context) error {
		return migrate.Permissions(a.DBConnectionFactory.GetDBMap, a.Cache)
	}})
	migrate.Add(sdk.Migration{Name: "WorkflowOldStruct", Release: "0.38.1", Mandatory: true, ExecFunc: func(ctx context.Context) error {
		return migrate.WorkflowRunOldModel(ctx, a.DBConnectionFactory.GetDBMap)
	}})
	migrate.Add(sdk.Migration{Name: "WorkflowNotification", Release: "0.38.1", Mandatory: true, ExecFunc: func(ctx context.Context) error {
		return migrate.WorkflowNotifications(a.Cache, a.DBConnectionFactory.GetDBMap)
	}})
	migrate.Add(sdk.Migration{Name: "CleanArtifactBuiltinActions", Release: "0.38.1", Mandatory: true, ExecFunc: func(ctx context.Context) error {
		return migrate.CleanArtifactBuiltinActions(a.Cache, a.DBConnectionFactory.GetDBMap)
	}})
	if os.Getenv("CDS_MIGRATE_ENABLE") == "true" {
		migrate.Add(sdk.Migration{Name: "MigrateActionDEPRECATEDGitClone", Release: "0.37.0", Mandatory: true, ExecFunc: func(ctx context.Context) error {
			return migrate.MigrateActionDEPRECATEDGitClone(a.mustDB, a.Cache)
		}})
	}
	// migrate.Add(sdk.Migration{Name: "GitClonePrivateKey", Release: "0.37.0", Mandatory: true, ExecFunc: func(ctx context.Context) error {
	// 	return migrate.GitClonePrivateKey(a.mustDB, a.Cache)
	// }})

	isFreshInstall, errF := version.IsFreshInstall(a.mustDB())
	if errF != nil {
		return sdk.WrapError(errF, "Unable to check if it's a fresh installation of CDS")
	}

	if isFreshInstall {
		if err := migrate.SaveAllMigrations(a.mustDB()); err != nil {
			return sdk.WrapError(err, "Cannot save all migrations to done")
		}
	} else {
		if sdk.VersionCurrent().Version != "" && !strings.HasPrefix(sdk.VersionCurrent().Version, "snapshot") {
			major, minor, _, errV := version.MaxVersion(a.mustDB())
			if errV != nil {
				return sdk.WrapError(errV, "Cannot fetch max version of CDS already started")
			}
			if major != 0 || minor != 0 {
				minSemverCompatible, _ := semver.Parse(migrate.MinCompatibleRelease)
				if major < minSemverCompatible.Major || (major == minSemverCompatible.Major && minor < minSemverCompatible.Minor) {
					return fmt.Errorf("there are some mandatory migrations which aren't done. Please check each changelog of CDS. Maybe you have skipped a release migration. The minimum compatible release is %s, please update to this release before", migrate.MinCompatibleRelease)
				}
			}
		}

		// Run all migrations in several goroutines
		migrate.Run(ctx, a.mustDB(), a.PanicDump())
	}

	// Init Services
	services.Initialize(ctx, a.DBConnectionFactory, a.PanicDump())

	externalServices := make([]sdk.ExternalService, 0, len(a.Config.Services))
	for _, s := range a.Config.Services {
		serv := sdk.ExternalService{
			Service: sdk.Service{
				Name:    s.Name,
				Type:    s.Type,
				HTTPURL: fmt.Sprintf("%s:%s%s", s.URL, s.Port, s.Path),
				GroupID: &group.SharedInfraGroup.ID,
			},
			HealthPort: s.HealthPort,
			HealthPath: s.HealthPath,
			HealthURL:  s.HealthURL,
			URL:        s.URL,
			Path:       s.Path,
			Port:       s.Port,
		}
		externalServices = append(externalServices, serv)
	}
	if err := services.InitExternal(a.mustDB, a.Cache, externalServices); err != nil {
		return fmt.Errorf("unable to init external service: %v", err)
	}
	sdk.GoRoutine(ctx, "pings-external-services",
		func(ctx context.Context) {
			services.Pings(ctx, a.mustDB, externalServices)
		}, a.PanicDump())

	// TODO: to delete after migration
	if os.Getenv("CDS_MIGRATE_GIT_CLONE") == "true" {
		go func() {
			if err := migrate.GitClonePrivateKey(a.mustDB, a.Cache); err != nil {
				log.Error("Bootstrap Error: %v", err)
			}
		}()
	}

	sdk.GoRoutine(ctx, "workflow.Initialize",
		func(ctx context.Context) {
			workflow.Initialize(ctx, a.DBConnectionFactory.GetDBMap, a.Cache, a.Config.URL.UI, a.Config.DefaultOS, a.Config.DefaultArch)
		}, a.PanicDump())
	sdk.GoRoutine(ctx, "PushInElasticSearch",
		func(ctx context.Context) {
			event.PushInElasticSearch(ctx, a.mustDB(), a.Cache)
		}, a.PanicDump())
	sdk.GoRoutine(ctx, "Metrics.pushInElasticSearch",
		func(ctx context.Context) {
			metrics.Init(ctx, a.DBConnectionFactory.GetDBMap)
		}, a.PanicDump())
	sdk.GoRoutine(ctx, "Purge",
		func(ctx context.Context) {
			purge.Initialize(ctx, a.Cache, a.DBConnectionFactory.GetDBMap, a.Metrics.WorkflowRunsMarkToDelete, a.Metrics.WorkflowRunsDeleted)
		}, a.PanicDump())

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port),
		Handler:        a.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		select {
		case <-ctx.Done():
			log.Warning("Cleanup SQL connections")
			s.Shutdown(ctx)
			a.DBConnectionFactory.Close()
			event.Publish(sdk.EventEngine{Message: "shutdown"}, nil)
			event.Close()
		}
	}()

	event.Publish(sdk.EventEngine{Message: fmt.Sprintf("started - listen on %d", a.Config.HTTP.Port)}, nil)

	go func() {
		//TLS is disabled for the moment. We need to serve TLS on HTTP too
		if err := grpcInit(a.DBConnectionFactory, a.Config.GRPC.Addr, a.Config.GRPC.Port, false, "", "", a.Config.Log.StepMaxSize); err != nil {
			log.Error("Cannot start GRPC server: %v", err)
		}
	}()

	if err := version.Upsert(a.mustDB()); err != nil {
		return sdk.WrapError(err, "Cannot upsert cds version")
	}

	log.Info("Starting CDS API HTTP Server on %s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port)
	if err := s.ListenAndServe(); err != nil {
		return fmt.Errorf("Cannot start HTTP server: %v", err)
	}

	return nil
}

const panicDumpTTL = 60 * 60 * 24 // 24 hours

func (a *API) PanicDump() func(s string) (io.WriteCloser, error) {
	return func(s string) (io.WriteCloser, error) {
		return cache.NewWriteCloser(a.Cache, cache.Key("api", "panic_dump", s), panicDumpTTL), nil
	}
}
