package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
)

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
	GetProjectIntegration() sdk.ProjectIntegration
	Status(ctx context.Context) sdk.MonitoringStatusLine
	Store(o Object, data io.ReadCloser) (string, error)
	Fetch(ctx context.Context, o Object) (io.ReadCloser, error)
	Delete(ctx context.Context, o Object) error
	DeleteContainer(ctx context.Context, containerPath string) error
	TemporaryURLSupported() bool
}

// DriverWithRedirect has to be implemented if your storage backend supports temp url
type DriverWithRedirect interface {
	// StoreURL returns a temporary url and a secret key to store an object
	StoreURL(o Object, contentType string) (url string, key string, err error)
	// FetchURL returns a temporary url and a secret key to fetch an object
	FetchURL(o Object) (url string, key string, err error)
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
	ProjectName     string
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
	Region     string
	BucketName string
	Prefix     string
	// Auth options, can provide a profile name, from environment or directly provide access keys
	AuthFromEnvironment bool
	SharedCredsFile     string
	Profile             string
	AccessKeyID         string
	SecretAccessKey     string
	SessionToken        string
	Endpoint            string //optional
	DisableSSL          bool   //optional
	ForcePathStyle      bool   //optional
}

// ConfigOptionsOpenstack is used by ConfigOptions
type ConfigOptionsOpenstack struct {
	Address         string
	Username        string
	Password        string
	Tenant          string
	Domain          string
	Region          string
	ContainerPrefix string
	DisableTempURL  bool
}

// ConfigOptionsFilesystem is used by ConfigOptions
type ConfigOptionsFilesystem struct {
	Basedir string
}

// GetDriver returns the storage driver, integration driver or sharedInfra shared otherwise
func GetDriver(ctx context.Context, db gorp.SqlExecutor, sharedStorage Driver, projectKey, integrationName string) (Driver, error) {
	if integrationName != sdk.DefaultStorageIntegrationName {
		storageDriver, err := initDriver(ctx, db, projectKey, integrationName)
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot load storage driver %s/%s", projectKey, integrationName)
		}
		return storageDriver, nil
	}
	return sharedStorage, nil
}

// initDriver init a storage driver from a project integration
func initDriver(ctx context.Context, db gorp.SqlExecutor, projectKey, integrationName string) (Driver, error) {
	projectIntegration, err := integration.LoadProjectIntegrationByNameWithClearPassword(ctx, db, projectKey, integrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot load projectIntegration %s/%s", projectKey, integrationName)
	}

	if !projectIntegration.Model.Storage {
		return nil, fmt.Errorf("projectIntegration.Model %t is not a storage integration", projectIntegration.Model.Storage)
	}

	switch projectIntegration.Model.Name {
	case sdk.AWSIntegrationModel:
		cfg := ConfigOptionsAWSS3{
			Region:          projectIntegration.Config["region"].Value,
			BucketName:      projectIntegration.Config["bucket_name"].Value,
			Prefix:          projectIntegration.Config["prefix"].Value,
			AccessKeyID:     projectIntegration.Config["access_key_id"].Value,
			SecretAccessKey: projectIntegration.Config["secret_access_key"].Value,
		}
		if endpoint := projectIntegration.Config["endpoint"].Value; endpoint != "" {
			cfg.Endpoint = endpoint
			cfg.DisableSSL, _ = strconv.ParseBool(projectIntegration.Config["disable_ssl"].Value)
			cfg.ForcePathStyle, _ = strconv.ParseBool(projectIntegration.Config["force_path_style"].Value)
		}
		return newS3Store(ctx, projectIntegration, cfg)
	case sdk.OpenstackIntegrationModel:
		return newSwiftStore(ctx, projectIntegration, ConfigOptionsOpenstack{
			Address:         projectIntegration.Config["address"].Value,
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

// Init initialise a new ArtifactStorage
func Init(c context.Context, cfg Config) (Driver, error) {
	switch cfg.Kind {
	case Openstack, Swift:
		return newSwiftStore(c, sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, cfg.Options.Openstack)
	case AWSS3:
		return newS3Store(c, sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, cfg.Options.AWSS3)
	case Filesystem:
		return newFilesystemStore(c, sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, cfg.Options.Filesystem)
	default:
		return nil, fmt.Errorf("Invalid flag --artifact-mode")
	}
}

func escape(container, object string) (string, string) {
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	object = url.QueryEscape(object)
	object = strings.Replace(object, "/", "-", -1)
	return container, object
}
