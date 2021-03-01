package api

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/audit"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/corpsso"
	"github.com/ovh/cds/engine/api/authentication/github"
	"github.com/ovh/cds/engine/api/authentication/gitlab"
	"github.com/ovh/cds/engine/api/authentication/ldap"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/authentication/oidc"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/version"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/websocket"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
)

// Configuration is the configuration structure for CDS API
type Configuration struct {
	Name string `toml:"name" comment:"Name of this CDS API Service\n Enter a name to enable this service" json:"name"`
	URL  struct {
		API string `toml:"api" default:"http://localhost:8081" json:"api"`
		UI  string `toml:"ui" default:"http://localhost:8080" json:"ui"`
	} `toml:"url" comment:"#####################\n CDS URLs Settings \n####################" json:"url"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen HTTP address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8081" json:"port"`
	} `toml:"http" json:"http"`
	Secrets struct {
		Key string `toml:"key" json:"-"`
	} `toml:"secrets" json:"secrets"`
	Database database.DBConfigurationWithEncryption `toml:"database" comment:"################################\n Postgresql Database settings \n###############################" json:"database"`
	Cache    struct {
		TTL   int `toml:"ttl" default:"60" json:"ttl"`
		Redis struct {
			Host     string `toml:"host" default:"localhost:6379" comment:"If your want to use a redis-sentinel based cluster, follow this syntax! <clustername>@sentinel1:26379,sentinel2:26379,sentinel3:26379" json:"host"`
			Password string `toml:"password" json:"-"`
		} `toml:"redis" comment:"Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup" json:"redis"`
	} `toml:"cache" comment:"######################\n CDS Cache Settings \n#####################" json:"cache"`
	Directories struct {
		Download string `toml:"download" default:"/var/lib/cds-engine" json:"download"`
	} `toml:"directories" json:"directories"`
	InternalServiceMesh struct {
		RequestSecondsTimeout int  `toml:"requestSecondsTimeout" json:"requestSecondsTimeout" default:"60"`
		InsecureSkipVerifyTLS bool `toml:"insecureSkipVerifyTLS" json:"insecureSkipVerifyTLS" default:"false"`
	} `toml:"internalServiceMesh" json:"internalServiceMesh"`
	Auth struct {
		DefaultGroup  string `toml:"defaultGroup" default:"" comment:"The default group is the group in which every new user will be granted at signup" json:"defaultGroup"`
		RSAPrivateKey string `toml:"rsaPrivateKey" default:"" comment:"The RSA Private Key used to sign and verify the JWT Tokens issued by the API \nThis is mandatory." json:"-"`
		LDAP          struct {
			Enabled         bool   `toml:"enabled" default:"false" json:"enabled"`
			SignupDisabled  bool   `toml:"signupDisabled" default:"false" json:"signupDisabled"`
			Host            string `toml:"host" json:"host"`
			Port            int    `toml:"port" default:"636" json:"port"`
			SSL             bool   `toml:"ssl" default:"true" json:"ssl"`
			RootDN          string `toml:"rootDN" default:"dc=myorganization,dc=com" json:"rootDN"`
			UserSearchBase  string `toml:"userSearchBase" default:"ou=people" json:"userSearchBase"`
			UserSearch      string `toml:"userSearch" default:"uid={0}" json:"userSearch"`
			UserFullname    string `toml:"userFullname" default:"{{.givenName}} {{.sn}}" json:"userFullname"`
			ManagerDN       string `toml:"managerDN" default:"cn=admin,dc=myorganization,dc=com" comment:"Define it if ldapsearch need to be authenticated" json:"managerDN"`
			ManagerPassword string `toml:"managerPassword" default:"SECRET_PASSWORD_MANAGER" comment:"Define it if ldapsearch need to be authenticated" json:"-"`
		} `toml:"ldap" json:"ldap"`
		Local struct {
			Enabled              bool   `toml:"enabled" default:"true" json:"enabled"`
			SignupDisabled       bool   `toml:"signupDisabled" default:"false" json:"signupDisabled"`
			SignupAllowedDomains string `toml:"signupAllowedDomains" default:"" comment:"Allow signup from selected domains only - comma separated. Example: your-domain.com,another-domain.com" commented:"true" json:"signupAllowedDomains"`
		} `toml:"local" json:"local"`
		CorporateSSO struct {
			MFASupportEnabled bool   `json:"mfa_support_enabled" default:"false" toml:"mfaSupportEnabled"`
			Enabled           bool   `json:"enabled" default:"false" toml:"enabled"`
			SignupDisabled    bool   `json:"signupDisabled" default:"false" toml:"signupDisabled"`
			MailDomain        string `json:"mailDomain" toml:"mailDomain"`
			RedirectMethod    string `json:"redirect_method" toml:"redirectMethod"`
			RedirectURL       string `json:"redirect_url" toml:"redirectURL"`
			Keys              struct {
				RequestSigningKey  string `json:"-" toml:"requestSigningKey"`
				TokenSigningKey    string `json:"-" toml:"tokenSigningKey"`
				TokenKeySigningKey struct {
					KeySigningKey   string `json:"public_signing_key" toml:"keySigningKey"`
					SigningKeyClaim string `json:"signing_key_claim" toml:"signingKeyClaim"`
				} `json:"-" toml:"tokenKeySigningKey"`
			} `json:"-" toml:"keys"`
		} `json:"corporate_sso" toml:"corporateSSO"`
		Github struct {
			Enabled        bool   `toml:"enabled" default:"false" json:"enabled"`
			SignupDisabled bool   `toml:"signupDisabled" default:"false" json:"signupDisabled"`
			URL            string `toml:"url" json:"url" default:"https://github.com" comment:"Github URL"`
			APIURL         string `toml:"apiUrl" json:"apiUrl" default:"https://api.github.com" comment:"Github API URL"`
			ClientID       string `toml:"clientId" json:"-" comment:"Github OAuth Client ID"`
			ClientSecret   string `toml:"clientSecret" json:"-" comment:"Github OAuth Client Secret"`
		} `toml:"github" json:"github" comment:"#######\n CDS <-> GitHub Auth. Documentation on https://ovh.github.io/cds/docs/integrations/github/github_authentication/ \n######"`
		Gitlab struct {
			Enabled        bool   `toml:"enabled" default:"false" json:"enabled"`
			SignupDisabled bool   `toml:"signupDisabled" default:"false" json:"signupDisabled"`
			URL            string `toml:"url" json:"url" default:"https://gitlab.com" comment:"GitLab URL"`
			ApplicationID  string `toml:"applicationID" json:"-" comment:"GitLab OAuth Application ID"`
			Secret         string `toml:"secret" json:"-" comment:"GitLab OAuth Application Secret"`
		} `toml:"gitlab" json:"gitlab" comment:"#######\n CDS <-> GitLab Auth. Documentation on https://ovh.github.io/cds/docs/integrations/gitlab/gitlab_authentication/ \n######"`
		OIDC struct {
			Enabled        bool   `toml:"enabled" default:"false" json:"enabled"`
			SignupDisabled bool   `toml:"signupDisabled" default:"false" json:"signupDisabled"`
			URL            string `toml:"url" json:"url" default:"" comment:"Open ID connect config URL"`
			ClientID       string `toml:"clientId" json:"-" comment:"OIDC Client ID"`
			ClientSecret   string `toml:"clientSecret" json:"-" comment:"OIDC Client Secret"`
		} `toml:"oidc" json:"oidc" comment:"#######\n CDS <-> Open ID Connect Auth. Documentation on https://ovh.github.io/cds/docs/integrations/openid-connect/ \n######"`
	} `toml:"auth" comment:"##############################\n CDS Authentication Settings# \n#############################" json:"auth"`
	SMTP struct {
		Disable               bool   `toml:"disable" default:"true" json:"disable" comment:"Set to false to enable the internal SMTP client. If false, emails will be displayed in CDS API Log."`
		Host                  string `toml:"host" json:"host" comment:"smtp host"`
		Port                  string `toml:"port" json:"port" comment:"smtp port"`
		ModeTLS               string `toml:"modeTLS" json:"modeTLS" default:"" comment:"possible values: empty, tls, starttls"`
		InsecureSkipVerifyTLS bool   `toml:"insecureSkipVerifyTLS" json:"insecureSkipVerifyTLS" default:"false" comment:"skip TLS verification with TLS / StartTLS mode"`
		User                  string `toml:"user" json:"user" comment:"smtp username"`
		Password              string `toml:"password" json:"-" comment:"smtp password"`
		From                  string `toml:"from" default:"no-reply@cds.local" json:"from" comment:"smtp from"`
	} `toml:"smtp" comment:"#####################\n# CDS SMTP Settings \n####################" json:"smtp"`
	Artifact struct {
		Mode  string `toml:"mode" default:"local" comment:"swift, awss3 or local" json:"mode"`
		Local struct {
			BaseDirectory string `toml:"baseDirectory" default:"/var/lib/cds-engine/artifacts" json:"baseDirectory"`
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
			Endpoint            string `toml:"endpoint" json:"endpoint" comment:"S3 API Endpoint (optional)" commented:"true"` //optional
			DisableSSL          bool   `toml:"disableSSL" json:"disableSSL" commented:"true"`                                  //optional
			ForcePathStyle      bool   `toml:"forcePathStyle" json:"forcePathStyle" commented:"true"`                          //optional
		} `toml:"awss3" json:"awss3"`
	} `toml:"artifact" comment:"Either filesystem local storage or Openstack Swift Storage are supported" json:"artifact"`
	Services    []sdk.ServiceConfiguration `toml:"services" comment:"###########################\n CDS Services Settings \n##########################" json:"services"`
	DefaultOS   string                     `toml:"defaultOS" default:"linux" comment:"if no model and os/arch is specified in your job's requirements then spawn worker on this operating system (example: freebsd, linux, windows)" json:"defaultOS"`
	DefaultArch string                     `toml:"defaultArch" default:"amd64" comment:"if no model and no os/arch is specified in your job's requirements then spawn worker on this architecture (example: amd64, arm, 386)" json:"defaultArch"`
	Graylog     struct {
		AccessToken string `toml:"accessToken" json:"-"`
		Stream      string `toml:"stream" json:"-"`
		URL         string `toml:"url" comment:"Example: http://localhost:9000" json:"url"`
	} `toml:"graylog" json:"graylog" comment:"###########################\n Graylog Search. \n When CDS API generates errors, you can fetch them with cdsctl. \n Examples: \n $ cdsctl admin errors get <error-id> \n $ cdsctl admin errors get 55f6e977-d39b-11e8-8513-0242ac110007 \n##########################"`
	Log struct {
		StepMaxSize    int64 `toml:"stepMaxSize" default:"15728640" comment:"Max step logs size in bytes (default: 15MB)" json:"stepMaxSize"`
		ServiceMaxSize int64 `toml:"serviceMaxSize" default:"15728640" comment:"Max service logs size in bytes (default: 15MB)" json:"serviceMaxSize"`
	} `toml:"log" json:"log" comment:"###########################\n Log settings.\n##########################"`
	Help struct {
		Content string `toml:"content" comment:"Help Content. Warning: this message could be view by anonymous user. Markdown accepted." json:"content" default:""`
		Error   string `toml:"error" comment:"Help displayed to user on each error. Warning: this message could be view by anonymous user. Markdown accepted." json:"error" default:""`
	} `toml:"help" comment:"######################\n 'Help' informations \n######################" json:"help"`
	Workflow struct {
		MaxRuns                int64  `toml:"maxRuns" comment:"Maximum of runs by workflow" json:"maxRuns" default:"255"`
		DefaultRetentionPolicy string `toml:"defaultRetentionPolicy" comment:"Default rule for workflow run retention policy, this rule can be overridden on each workflow.\n Example: 'return run_days_before < 365' keeps runs for one year." json:"defaultRetentionPolicy" default:"return run_days_before < 365"`
	} `toml:"workflow" comment:"######################\n 'Workflow' global configuration \n######################" json:"workflow"`
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

func (*API) Init(i interface{}) (cdsclient.ServiceConfig, error) {
	return cdsclient.ServiceConfig{}, nil
}

// Service returns an instance of sdk.Service for the API
func (*API) Service() sdk.Service {
	return sdk.Service{
		LastHeartbeat: time.Time{},
		CanonicalService: sdk.CanonicalService{
			Type: sdk.TypeAPI,
		},
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
	WSBroker            *websocket.Broker
	WSServer            *websocketServer
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
		DatabaseConns            *stats.Int64Measure
	}
	AuthenticationDrivers map[sdk.AuthConsumerType]sdk.AuthDriver
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

	a.Common.ServiceType = sdk.TypeAPI
	a.Common.ServiceName = a.Config.Name
	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (a *API) CheckConfiguration(config interface{}) error {
	aConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid API configuration")
	}

	if aConfig.Name == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	if aConfig.URL.API == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	if aConfig.URL.UI != "" {
		if _, err := url.Parse(aConfig.URL.UI); err != nil {
			return fmt.Errorf("Invalid given UI URL")
		}
	}

	if aConfig.Directories.Download == "" {
		return fmt.Errorf("Invalid download directory (empty)")
	}

	if ok, err := sdk.DirectoryExists(aConfig.Directories.Download); !ok {
		if err := os.MkdirAll(aConfig.Directories.Download, os.FileMode(0700)); err != nil {
			return fmt.Errorf("Unable to create directory %s: %v", aConfig.Directories.Download, err)
		}
		log.Info(context.Background(), "Directory %s has been created", aConfig.Directories.Download)
	} else if err != nil {
		return fmt.Errorf("Invalid download directory %s: %v", aConfig.Directories.Download, err)
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
		if ok, err := sdk.DirectoryExists(aConfig.Artifact.Local.BaseDirectory); !ok {
			if err := os.MkdirAll(aConfig.Artifact.Local.BaseDirectory, os.FileMode(0700)); err != nil {
				return fmt.Errorf("Unable to create directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
			}
			log.Info(context.Background(), "Directory %s has been created", aConfig.Artifact.Local.BaseDirectory)
		} else if err != nil {
			return fmt.Errorf("Invalid artifact local base directory %s: %v", aConfig.Artifact.Local.BaseDirectory, err)
		}
	}

	if len(aConfig.Secrets.Key) != 32 {
		return fmt.Errorf("Invalid secret key. It should be 32 bits (%d)", len(aConfig.Secrets.Key))
	}

	if aConfig.DefaultArch == "" {
		log.Warn(context.Background(), `You should add a default architecture in your configuration (example: defaultArch: "amd64"). It means if there is no model and os/arch requirement on your job then spawn on a worker based on this architecture`)
	}
	if aConfig.DefaultOS == "" {
		log.Warn(context.Background(), `You should add a default operating system in your configuration (example: defaultOS: "linux"). It means if there is no model and os/arch requirement on your job then spawn on a worker based on this OS`)
	}

	if (aConfig.DefaultOS == "" && aConfig.DefaultArch != "") || (aConfig.DefaultOS != "" && aConfig.DefaultArch == "") {
		return fmt.Errorf("You can't specify just defaultArch without defaultOS in your configuration and vice versa")
	}

	if aConfig.Auth.RSAPrivateKey == "" {
		return errors.New("invalid given authentication rsa private key")
	}

	return nil
}

type StartupConfigConsumerType string

const (
	StartupConfigConsumerTypeUI            StartupConfigConsumerType = "ui"
	StartupConfigConsumerTypeHatchery      StartupConfigConsumerType = "hatchery"
	StartupConfigConsumerTypeHooks         StartupConfigConsumerType = "hooks"
	StartupConfigConsumerTypeRepositories  StartupConfigConsumerType = "repositories"
	StartupConfigConsumerTypeDBMigrate     StartupConfigConsumerType = "db-migrate"
	StartupConfigConsumerTypeVCS           StartupConfigConsumerType = "vcs"
	StartupConfigConsumerTypeCDN           StartupConfigConsumerType = "cdn"
	StartupConfigConsumerTypeCDNStorageCDS StartupConfigConsumerType = "cdn-storage-cds"
	StartupConfigConsumerTypeElasticsearch StartupConfigConsumerType = "elasticsearch"
)

type StartupConfig struct {
	Consumers []StartupConfigConsumer `json:"consumers"`
	IAT       int64                   `json:"iat"`
}

type StartupConfigConsumer struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Type        StartupConfigConsumerType `json:"type"`
}

// Serve will start the http api server
func (a *API) Serve(ctx context.Context) error {
	log.Info(ctx, "Starting CDS API Server %s", sdk.VERSION)

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
		log.Error(ctx, "engine is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", a.Config.Directories.Download)
	}
	if !hasCtl {
		log.Error(ctx, "cdsctl is unavailable for download, this may lead to a poor user experience. Please check your configuration file or the %s directory", a.Config.Directories.Download)
	}
	if !hasWorker {
		// If no worker, let's exit because CDS for run anything
		log.Error(ctx, "worker is unavailable for download. Please check your configuration file or the %s directory", a.Config.Directories.Download)
		return errors.New("worker binary unavailable")
	}

	// Initialize the jwt layer
	if err := authentication.Init(a.ServiceName, []byte(a.Config.Auth.RSAPrivateKey)); err != nil {
		return sdk.WrapError(err, "unable to initialize the JWT Layer")
	}

	// Intialize service mesh httpclient
	if a.Config.InternalServiceMesh.RequestSecondsTimeout == 0 {
		a.Config.InternalServiceMesh.RequestSecondsTimeout = 60
	}
	services.HTTPClient = cdsclient.NewHTTPClient(
		time.Duration(a.Config.InternalServiceMesh.RequestSecondsTimeout)*time.Second,
		a.Config.InternalServiceMesh.InsecureSkipVerifyTLS,
	)

	// Initialize mail package
	log.Info(ctx, "Initializing mail driver...")
	mail.Init(a.Config.SMTP.User,
		a.Config.SMTP.Password,
		a.Config.SMTP.From,
		a.Config.SMTP.Host,
		a.Config.SMTP.Port,
		a.Config.SMTP.ModeTLS,
		a.Config.SMTP.InsecureSkipVerifyTLS,
		a.Config.SMTP.Disable)

	//Initialize artifacts storage
	log.Info(ctx, "Initializing %s objectstore...", a.Config.Artifact.Mode)
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
				Endpoint:            a.Config.Artifact.AWSS3.Endpoint,
				DisableSSL:          a.Config.Artifact.AWSS3.DisableSSL,
				ForcePathStyle:      a.Config.Artifact.AWSS3.ForcePathStyle,
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

	// API Storage will be a public integration
	var err error
	a.SharedStorage, err = objectstore.Init(ctx, cfg)
	if err != nil {
		return fmt.Errorf("cannot initialize storage: %v", err)
	}

	log.Info(ctx, "Initializing database connection...")
	//Intialize database
	a.DBConnectionFactory, err = database.Init(
		ctx,
		a.Config.Database.User,
		a.Config.Database.Role,
		a.Config.Database.Password,
		a.Config.Database.Name,
		a.Config.Database.Schema,
		a.Config.Database.Host,
		a.Config.Database.Port,
		a.Config.Database.SSLMode,
		a.Config.Database.ConnectTimeout,
		a.Config.Database.Timeout,
		a.Config.Database.MaxConn)
	if err != nil {
		return fmt.Errorf("cannot connect to database: %v", err)
	}

	log.Info(ctx, "Setting up database keys...")
	encryptionKeyConfig := a.Config.Database.EncryptionKey.GetKeys(gorpmapper.KeyEcnryptionIdentifier)
	signatureKeyConfig := a.Config.Database.SignatureKey.GetKeys(gorpmapper.KeySignIdentifier)
	if err := gorpmapping.ConfigureKeys(&signatureKeyConfig, &encryptionKeyConfig); err != nil {
		return fmt.Errorf("cannot setup database keys: %v", err)
	}

	// Init dao packages
	featureflipping.Init(gorpmapping.Mapper)

	log.Info(ctx, "Initializing redis cache on %s...", a.Config.Cache.Redis.Host)
	// Init the cache
	a.Cache, err = cache.New(
		a.Config.Cache.Redis.Host,
		a.Config.Cache.Redis.Password,
		a.Config.Cache.TTL)
	if err != nil {
		return sdk.WrapError(err, "cannot connect to cache store")
	}

	log.Info(ctx, "Initializing HTTP router")
	a.GoRoutines = sdk.NewGoRoutines()
	a.Router = &Router{
		Mux:        mux.NewRouter(),
		Background: ctx,
	}
	a.InitRouter()
	if err := a.initWebsocket(event.DefaultPubSubKey); err != nil {
		return err
	}
	if err := InitRouterMetrics(ctx, a); err != nil {
		log.Error(ctx, "unable to init router metrics: %v", err)
	}

	log.Info(ctx, "Initializing Metrics")
	if err := a.initMetrics(ctx); err != nil {
		log.Error(ctx, "unable to init api metrics: %v", err)
	}

	// Intialize notification package
	notification.Init(a.Config.URL.UI)

	log.Info(ctx, "Initializing Authentication drivers...")
	a.AuthenticationDrivers = make(map[sdk.AuthConsumerType]sdk.AuthDriver)

	a.AuthenticationDrivers[sdk.ConsumerBuiltin] = builtin.NewDriver()
	if a.Config.Auth.Local.Enabled {
		a.AuthenticationDrivers[sdk.ConsumerLocal] = local.NewDriver(
			ctx,
			a.Config.Auth.Local.SignupDisabled,
			a.Config.URL.UI,
			a.Config.Auth.Local.SignupAllowedDomains,
		)
	}

	if a.Config.Auth.LDAP.Enabled {
		a.AuthenticationDrivers[sdk.ConsumerLDAP], err = ldap.NewDriver(
			ctx,
			a.Config.Auth.LDAP.SignupDisabled,
			ldap.Config{
				Host:            a.Config.Auth.LDAP.Host,
				Port:            a.Config.Auth.LDAP.Port,
				SSL:             a.Config.Auth.LDAP.SSL,
				RootDN:          a.Config.Auth.LDAP.RootDN,
				UserSearchBase:  a.Config.Auth.LDAP.UserSearchBase,
				UserSearch:      a.Config.Auth.LDAP.UserSearch,
				UserFullname:    a.Config.Auth.LDAP.UserFullname,
				ManagerDN:       a.Config.Auth.LDAP.ManagerDN,
				ManagerPassword: a.Config.Auth.LDAP.ManagerPassword,
			},
		)
		if err != nil {
			return err
		}
	}
	if a.Config.Auth.Github.Enabled {
		a.AuthenticationDrivers[sdk.ConsumerGithub] = github.NewDriver(
			a.Config.Auth.Github.SignupDisabled,
			a.Config.URL.UI,
			a.Config.Auth.Github.URL,
			a.Config.Auth.Github.APIURL,
			a.Config.Auth.Github.ClientID,
			a.Config.Auth.Github.ClientSecret,
		)
	}
	if a.Config.Auth.Gitlab.Enabled {
		a.AuthenticationDrivers[sdk.ConsumerGitlab] = gitlab.NewDriver(
			a.Config.Auth.Gitlab.SignupDisabled,
			a.Config.URL.UI,
			a.Config.Auth.Gitlab.URL,
			a.Config.Auth.Gitlab.ApplicationID,
			a.Config.Auth.Gitlab.Secret,
		)
	}
	if a.Config.Auth.OIDC.Enabled {
		a.AuthenticationDrivers[sdk.ConsumerOIDC], err = oidc.NewDriver(
			a.Config.Auth.OIDC.SignupDisabled,
			a.Config.URL.UI,
			a.Config.Auth.OIDC.URL,
			a.Config.Auth.OIDC.ClientID,
			a.Config.Auth.OIDC.ClientSecret,
		)
		if err != nil {
			return err
		}
	}

	if a.Config.Auth.CorporateSSO.Enabled {
		driverConfig := corpsso.Config{
			MailDomain:        a.Config.Auth.CorporateSSO.MailDomain,
			MFASupportEnabled: a.Config.Auth.CorporateSSO.MFASupportEnabled,
		}
		driverConfig.Request.Keys.RequestSigningKey = a.Config.Auth.CorporateSSO.Keys.RequestSigningKey
		driverConfig.Request.RedirectMethod = a.Config.Auth.CorporateSSO.RedirectMethod
		driverConfig.Request.RedirectURL = a.Config.Auth.CorporateSSO.RedirectURL
		driverConfig.Token.SigningKey = a.Config.Auth.CorporateSSO.Keys.TokenSigningKey
		driverConfig.Token.KeySigningKey.KeySigningKey = a.Config.Auth.CorporateSSO.Keys.TokenKeySigningKey.KeySigningKey
		driverConfig.Token.KeySigningKey.SigningKeyClaim = a.Config.Auth.CorporateSSO.Keys.TokenKeySigningKey.SigningKeyClaim

		a.AuthenticationDrivers[sdk.ConsumerCorporateSSO] = corpsso.NewDriver(driverConfig)
	}

	log.Info(ctx, "Initializing event broker...")
	if err := event.Initialize(ctx, a.mustDB(), a.Cache); err != nil {
		log.Error(ctx, "error while initializing event system: %s", err)
	}

	a.GoRoutines.Run(ctx, "event.dequeue", func(ctx context.Context) {
		event.DequeueEvent(ctx, a.mustDB())
	})

	log.Info(ctx, "Initializing internal routines...")
	a.GoRoutines.Run(ctx, "maintenance.Subscribe", func(ctx context.Context) {
		if err := a.listenMaintenance(ctx); err != nil {
			log.Error(ctx, "error while initializing listen maintenance routine: %s", err)
		}
	})

	a.GoRoutines.Exec(ctx, "workermodel.Initialize", func(ctx context.Context) {
		if err := workermodel.Initialize(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Cache); err != nil {
			log.Error(ctx, "error while initializing worker models routine: %s", err)
		}
	})
	a.GoRoutines.Run(ctx, "worker.Initialize", func(ctx context.Context) {
		if err := worker.Initialize(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Cache); err != nil {
			log.Error(ctx, "error while initializing workers routine: %s", err)
		}
	})
	a.GoRoutines.Run(ctx, "action.ComputeAudit", func(ctx context.Context) {
		chanEvent := make(chan sdk.Event)
		event.Subscribe(chanEvent)
		action.ComputeAudit(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), chanEvent)
	})
	a.GoRoutines.Run(ctx, "audit.ComputePipelineAudit", func(ctx context.Context) {
		audit.ComputePipelineAudit(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
	})
	a.GoRoutines.Run(ctx, "audit.ComputeWorkflowAudit", func(ctx context.Context) {
		audit.ComputeWorkflowAudit(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
	})
	a.GoRoutines.Run(ctx, "auditCleanerRoutine(ctx", func(ctx context.Context) {
		auditCleanerRoutine(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
	})
	a.GoRoutines.Run(ctx, "repositoriesmanager.ReceiveEvents", func(ctx context.Context) {
		repositoriesmanager.ReceiveEvents(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Cache)
	})
	a.GoRoutines.Run(ctx, "services.KillDeadServices", func(ctx context.Context) {
		services.KillDeadServices(ctx, a.mustDB)
	})
	a.GoRoutines.Run(ctx, "broadcast.Initialize", func(ctx context.Context) {
		broadcast.Initialize(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
	})
	a.GoRoutines.Run(ctx, "api.serviceAPIHeartbeat", func(ctx context.Context) {
		a.serviceAPIHeartbeat(ctx)
	})
	a.GoRoutines.Run(ctx, "authentication.SessionCleaner", func(ctx context.Context) {
		authentication.SessionCleaner(ctx, a.mustDB, 10*time.Second)
	})
	a.GoRoutines.Run(ctx, "api.WorkflowRunCraft", func(ctx context.Context) {
		a.WorkflowRunCraft(ctx, 100*time.Millisecond)
	})

	migrate.Add(ctx, sdk.Migration{Name: "RunsSecrets", Release: "0.47.0", Blocker: false, Automatic: true, ExecFunc: func(ctx context.Context) error {
		return migrate.RunsSecrets(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
	}})

	isFreshInstall, err := version.IsFreshInstall(a.mustDB())
	if err != nil {
		return sdk.WrapError(err, "Unable to check if it's a fresh installation of CDS")
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
		migrate.Run(ctx, a.mustDB())
	}

	log.Info(ctx, "Bootstrapping database...")
	defaultValues := sdk.DefaultValues{
		DefaultGroupName: a.Config.Auth.DefaultGroup,
	}
	if err := bootstrap.InitiliazeDB(ctx, defaultValues, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)); err != nil {
		return fmt.Errorf("cannot setup databases: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow hook models: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowOutgoingHookModels(a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow outgoing hook models: %v", err)
	}

	if err := integration.CreateBuiltinModels(a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper)()); err != nil {
		return fmt.Errorf("cannot setup integrations: %v", err)
	}

	pubKey, err := jws.ExportPublicKey(authentication.GetSigningKey())
	if err != nil {
		return sdk.WrapError(err, "Unable to export public signing key")
	}

	log.Info(ctx, "API Public Key: \n%s", string(pubKey))

	// Init Services
	services.Initialize(ctx, a.DBConnectionFactory, a.GoRoutines)

	externalServices := make([]services.ExternalService, 0, len(a.Config.Services))
	for _, s := range a.Config.Services {
		log.Info(ctx, "Managing external service %s %s", s.Name, s.Type)
		serv := services.ExternalService{
			Service: sdk.Service{
				CanonicalService: sdk.CanonicalService{
					Name:    s.Name,
					Type:    s.Type,
					HTTPURL: fmt.Sprintf("%s:%s%s", s.URL, s.Port, s.Path),
				},
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

	if err := services.InitExternal(ctx, a.mustDB(), externalServices); err != nil {
		return fmt.Errorf("unable to init external service: %+v", err)
	}
	a.GoRoutines.Run(ctx, "pings-external-services",
		func(ctx context.Context) {
			services.Pings(ctx, a.mustDB, externalServices)
		})
	a.GoRoutines.Run(ctx, "workflow.Initialize",
		func(ctx context.Context) {
			workflow.SetMaxRuns(a.Config.Workflow.MaxRuns)
			workflow.Initialize(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Cache, a.Config.URL.UI, a.Config.DefaultOS, a.Config.DefaultArch, a.Config.Log.StepMaxSize)
		})
	a.GoRoutines.Run(ctx, "PushInElasticSearch",
		func(ctx context.Context) {
			event.PushInElasticSearch(ctx, a.mustDB(), a.Cache)
		})
	a.GoRoutines.Run(ctx, "Metrics.pushInElasticSearch",
		func(ctx context.Context) {
			metrics.Init(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper))
		})

	// init purge
	if err := purge.SetDefaultRunRetentionPolicy(a.Config.Workflow.DefaultRetentionPolicy); err != nil {
		return err
	}
	a.GoRoutines.Run(ctx, "Purge-MarkRuns",
		func(ctx context.Context) {
			purge.MarkRunsAsDelete(ctx, a.Cache, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Metrics.WorkflowRunsMarkToDelete)
		})
	a.GoRoutines.Run(ctx, "Purge-Runs",
		func(ctx context.Context) {
			purge.WorkflowRuns(ctx, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.SharedStorage, a.Metrics.WorkflowRunsMarkToDelete, a.Metrics.WorkflowRunsDeleted)
		})
	a.GoRoutines.Run(ctx, "Purge-Workflow",
		func(ctx context.Context) {
			purge.Workflow(ctx, a.Cache, a.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), a.Metrics.WorkflowRunsMarkToDelete)
		})

	// Check maintenance on redis
	if _, err := a.Cache.Get(sdk.MaintenanceAPIKey, &a.Maintenance); err != nil {
		return err
	}

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port),
		Handler:        a.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		<-ctx.Done()
		log.Warn(ctx, "Cleanup SQL connections")
		s.Shutdown(ctx)               // nolint
		a.DBConnectionFactory.Close() // nolint
		event.Publish(ctx, sdk.EventEngine{Message: "shutdown"}, nil)
		event.Close(ctx)
	}()

	event.Publish(ctx, sdk.EventEngine{Message: fmt.Sprintf("started - listen on %d", a.Config.HTTP.Port)}, nil)

	if err := version.Upsert(a.mustDB()); err != nil {
		return sdk.WrapError(err, "Cannot upsert cds version")
	}

	// Dump heap to objecstore on SIGINFO
	siginfoChan := make(chan os.Signal, 1)
	signal.Notify(siginfoChan, sdk.SIGINFO)
	go func() {
		<-siginfoChan
		signal.Stop(siginfoChan)
		var buffer = new(bytes.Buffer)
		pprof.Lookup("heap").WriteTo(buffer, 1)
		var heapProfile = heapProfile{uuid: sdk.UUID()}
		s, err := a.SharedStorage.Store(heapProfile, ioutil.NopCloser(buffer))
		if err != nil {
			log.Error(ctx, "unable to upload heap profile: %v", err)
			return
		}
		log.Error(ctx, "api> heap dump uploaded to %s", s)
	}()

	log.Info(ctx, "Starting CDS API HTTP Server on %s:%d", a.Config.HTTP.Addr, a.Config.HTTP.Port)
	if err := s.ListenAndServe(); err != nil {
		return fmt.Errorf("Cannot start HTTP server: %v", err)
	}

	return nil
}

// SetCookieSession on given response writter, automatically add domain and path based on api config.
// This will returns a cookie with no expiration date that should be dropped by browser when closed.
func (a *API) SetCookieSession(w http.ResponseWriter, name, value string) {
	a.setCookie(w, &http.Cookie{
		Name:  name,
		Value: value,
	})
}

// SetCookie on given response writter, automatically add domain and path based on api config.
func (a *API) SetCookie(w http.ResponseWriter, name, value string, expires time.Time) {
	a.setCookie(w, &http.Cookie{
		Name:    name,
		Value:   value,
		Expires: expires,
	})
}

// UnsetCookie on given response writter, automatically add domain and path based on api config.
func (a *API) UnsetCookie(w http.ResponseWriter, name string) {
	a.setCookie(w, &http.Cookie{
		Name:   name,
		Value:  "",
		MaxAge: -1,
	})
}

func (a *API) setCookie(w http.ResponseWriter, c *http.Cookie) {
	if a.Config.URL.UI != "" {
		// ignore parse error, this have been checked at service start
		uiURL, _ := url.Parse(a.Config.URL.UI)
		c.Path = uiURL.Path
		if c.Path == "" {
			c.Path = "/"
		}
	}
	http.SetCookie(w, c)
}

type heapProfile struct {
	uuid string
}

var _ objectstore.Object = new(heapProfile)

func (p heapProfile) GetName() string {
	return p.uuid
}
func (p heapProfile) GetPath() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("api-heap-profile-%d-%s", time.Now().Unix(), hostname)
}
