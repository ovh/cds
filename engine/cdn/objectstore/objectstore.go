package objectstore

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
	GetProjectIntegration() sdk.ProjectIntegration
	Status() sdk.MonitoringStatusLine
	Store(ctx context.Context, o Object, data io.ReadCloser) (string, error)
	ServeStaticFiles(o Object, entrypoint string, data io.ReadCloser) (string, error)
	Fetch(ctx context.Context, o Object) (io.ReadCloser, error)
	Delete(ctx context.Context, o Object) error
	TemporaryURLSupported() bool
}

// DriverWithRedirect has to be implemented if your storage backend supports temp url
type DriverWithRedirect interface {
	// StoreURL returns a temporary url and a secret key to store an object
	StoreURL(o Object, contentType string) (url string, key string, err error)
	// FetchURL returns a temporary url and a secret key to fetch an object
	FetchURL(o Object) (url string, key string, err error)
	// ServeStaticFilesURL returns a temporary url and a secret key to serve static files in a container
	ServeStaticFilesURL(o Object, entrypoint string) (string, string, error)
}

// Kind will define const defining all supported objecstore drivers
type Kind int

// These are the defined objecstore drivers
const (
	Openstack Kind = iota
	Filesystem
	Swift
	AWSS3
)

// Config represents all the configuration for all objectstore drivers
type Config struct {
	IntegrationName string
	Kind            Kind
	Options         ConfigOptions
}

// ConfigOptions is used by Config
type ConfigOptions struct {
	AWSS3      ConfigOptionsAWSS3
	Openstack  ConfigOptionsOpenstack
	Filesystem ConfigOptionsFilesystem
}

// ConfigOptionsAWSS3 is used by ConfigOptions
type ConfigOptionsAWSS3 struct {
	BucketName string `toml:"bucketName" json:"bucketName" comment:"Name of the S3 bucket to use when storing artifacts"`
	Region     string `toml:"region" json:"region" default:"us-east-1" comment:"The AWS region"`
	Prefix     string `toml:"prefix" json:"prefix" comment:"A subfolder of the bucket to store objects in, if left empty will store at the root of the bucket"`
	// Auth options, can provide a profile name, from environment or directly provide access keys
	AuthFromEnvironment bool   `toml:"authFromEnv" json:"authFromEnv" default:"false" comment:"Pull S3 auth information from env vars AWS_SECRET_ACCESS_KEY and AWS_SECRET_KEY_ID"`
	SharedCredsFile     string `toml:"sharedCredsFile" json:"sharedCredsFile" comment:"The path for the AWS credential file, used with profile"`
	Profile             string `toml:"profile" json:"profile" comment:"The profile within the AWS credentials file to use"`
	AccessKeyID         string `toml:"accessKeyId" json:"accessKeyId" comment:"A static AWS Secret Key ID"`
	SecretAccessKey     string `toml:"secretAccessKey" json:"-" comment:"A static AWS Secret Access Key"`
	SessionToken        string `toml:"sessionToken" json:"-" comment:"A static AWS session token"`
	Endpoint            string `toml:"endpoint" json:"endpoint" comment:"S3 API Endpoint (optional)" commented:"true"` //optional
	DisableSSL          bool   `toml:"disableSSL" json:"disableSSL" commented:"true"`                                  //optional
	ForcePathStyle      bool   `toml:"forcePathStyle" json:"forcePathStyle" commented:"true"`                          //optional
}

// ConfigOptionsOpenstack is used by ConfigOptions
type ConfigOptionsOpenstack struct {
	URL             string `toml:"url" comment:"Authentication Endpoint, generally value of $OS_AUTH_URL" json:"url"`
	Username        string `toml:"username" comment:"Openstack Username, generally value of $OS_USERNAME" json:"username"`
	Password        string `toml:"password" comment:"Openstack Password, generally value of $OS_PASSWORD" json:"-"`
	Tenant          string `toml:"tenant" comment:"Openstack Tenant, generally value of $OS_TENANT_NAME, v2 auth only" json:"tenant"`
	Domain          string `toml:"domain" comment:"Openstack Domain, generally value of $OS_DOMAIN_NAME, v3 auth only" json:"domain"`
	Region          string `toml:"region" comment:"Region, generally value of $OS_REGION_NAME" json:"region"`
	ContainerPrefix string `toml:"containerPrefix" comment:"Use if your want to prefix containers for CDS Artifacts" json:"containerPrefix"`
	DisableTempURL  bool   `toml:"disableTempURL" default:"false" commented:"true" comment:"True if you want to disable Temporary URL in file upload" json:"disableTempURL"`
}

// ConfigOptionsFilesystem is used by ConfigOptions
type ConfigOptionsFilesystem struct {
	BaseDirectory string `toml:"baseDirectory" default:"/tmp/cds/artifacts" json:"baseDirectory"`
}

// InitDriver init a storage driver from a project integration
func InitDriver(projectIntegration sdk.ProjectIntegration) (Driver, error) {
	if !projectIntegration.Model.Storage {
		return nil, fmt.Errorf("projectIntegration.Model %t is not a storage integration", projectIntegration.Model.Storage)
	}

	switch projectIntegration.Model.Name {
	case sdk.AWSIntegrationModel:
		return newS3Store(projectIntegration, ConfigOptionsAWSS3{
			Region:          projectIntegration.Config["region"].Value,
			BucketName:      projectIntegration.Config["bucket_name"].Value,
			Prefix:          projectIntegration.Config["prefix"].Value,
			AccessKeyID:     projectIntegration.Config["access_key_id"].Value,
			SecretAccessKey: projectIntegration.Config["secret_access_key"].Value,
			Endpoint:        projectIntegration.Config["endpoint"].Value,
			DisableSSL:      projectIntegration.Config["disable_ssl"].Value == "true",
			ForcePathStyle:  projectIntegration.Config["force_path_style"].Value == "true",
		})
	case sdk.OpenstackIntegrationModel:
		return newSwiftStore(projectIntegration, ConfigOptionsOpenstack{
			URL:             projectIntegration.Config["address"].Value,
			Region:          projectIntegration.Config["region"].Value,
			Tenant:          projectIntegration.Config["tenant_name"].Value,
			Domain:          projectIntegration.Config["domain"].Value,
			Username:        projectIntegration.Config["username"].Value,
			Password:        projectIntegration.Config["password"].Value,
			ContainerPrefix: projectIntegration.Config["storage_container_prefix"].Value,
			DisableTempURL:  projectIntegration.Config["storage_temporary_url_supported"].Value == "false",
		})
	default:
		return nil, fmt.Errorf("Invalid Integration %s", projectIntegration.Model.Name)
	}
}

// Init initialise a new driver
func Init(c context.Context, cfg Config) (Driver, error) {
	switch cfg.Kind {
	case Openstack, Swift:
		return newSwiftStore(sdk.ProjectIntegration{Name: cfg.IntegrationName}, cfg.Options.Openstack)
	case AWSS3:
		return newS3Store(sdk.ProjectIntegration{Name: cfg.IntegrationName}, cfg.Options.AWSS3)
	case Filesystem:
		return NewFilesystemStore(sdk.ProjectIntegration{Name: cfg.IntegrationName}, cfg.Options.Filesystem)
	default:
		return nil, fmt.Errorf("Invalid flag --artifact-mode")
	}
}
