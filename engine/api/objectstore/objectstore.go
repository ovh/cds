package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ncw/swift"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
)

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
	GetProjectIntegration() sdk.ProjectIntegration
	Status() sdk.MonitoringStatusLine
	Store(o Object, data io.ReadCloser) (string, error)
	ServeStaticFiles(o Object, entrypoint string, data io.ReadCloser) (string, error)
	Fetch(o Object) (io.ReadCloser, error)
	Delete(o Object) error
	TemporaryURLSupported() bool
}

// DriverWithRedirect has to be implemented if your storage backend supports temp url
type DriverWithRedirect interface {
	// StoreURL returns a temporary url and a secret key to store an object
	StoreURL(o Object) (url string, key string, err error)
	// FetchURL returns a temporary url and a secret key to fetch an object
	FetchURL(o Object) (url string, key string, err error)
	// ServeStaticFilesURL returns a temporary url and a secret key to serve static files in a container
	ServeStaticFilesURL(o Object, entrypoint string) (string, string, error)
	// GetPublicURL returns a public url to fetch an object (check your object ACLs before)
	GetPublicURL(o Object) (url string, err error)
}

// Kind will define const defining all supported objecstore drivers
type Kind int

// These are the defined objecstore drivers
const (
	Openstack Kind = iota
	Filesystem
	Swift
)

// Config represents all the configuration for all objecstore drivers
type Config struct {
	IntegrationName string
	ProjectName     string
	Kind            Kind
	Options         ConfigOptions
}

// ConfigOptions is used by Config
type ConfigOptions struct {
	Openstack  ConfigOptionsOpenstack
	Filesystem ConfigOptionsFilesystem
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

// InitDriver init a storage driver from a project integration
func InitDriver(db gorp.SqlExecutor, projectKey, integrationName string) (Driver, error) {
	projectIntegration, err := integration.LoadProjectIntegrationByName(db, projectKey, integrationName, false)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot load projectIntegration %s/%s", projectKey, integrationName)
	}

	if projectIntegration.Model.Storage == false {
		return nil, fmt.Errorf("projectIntegration.Model %t is not a storage integration", projectIntegration.Model.Storage)
	}

	if projectIntegration.Model.Name == sdk.OpenstackIntegrationModel {
		s := SwiftStore{
			Connection: swift.Connection{
				AuthUrl:  projectIntegration.Config["address"].Value,
				Region:   projectIntegration.Config["region"].Value,
				Tenant:   projectIntegration.Config["tenant"].Value,
				Domain:   projectIntegration.Config["domain"].Value,
				UserName: projectIntegration.Config["user"].Value,
				ApiKey:   projectIntegration.Config["password"].Value,
			},
			containerprefix:    projectIntegration.Config["storage_container_prefix"].Value,
			disableTempURL:     projectIntegration.Config["storage_temporary_url_supported"].Value == "true",
			projectIntegration: projectIntegration,
		}

		if err := s.Authenticate(); err != nil {
			return nil, sdk.WrapError(err, "Unable to authenticate")
		}
		return &s, nil
	}

	return nil, fmt.Errorf("Invalid Integration %s", projectIntegration.Model.Name)
}

// Init initialise a new ArtifactStorage
func Init(c context.Context, cfg Config) (Driver, error) {
	switch cfg.Kind {
	case Openstack, Swift:
		s := SwiftStore{
			Connection: swift.Connection{
				AuthUrl:  cfg.Options.Openstack.Address,
				Region:   cfg.Options.Openstack.Region,
				Tenant:   cfg.Options.Openstack.Tenant,
				Domain:   cfg.Options.Openstack.Domain,
				UserName: cfg.Options.Openstack.Username,
				ApiKey:   cfg.Options.Openstack.Password,
			},
			containerprefix:    cfg.Options.Openstack.ContainerPrefix,
			disableTempURL:     cfg.Options.Openstack.DisableTempURL,
			projectIntegration: sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName},
		}
		if err := s.Authenticate(); err != nil {
			return nil, sdk.WrapError(err, "Unable to authenticate on swift storage")
		}
		return &s, nil
	case Filesystem:
		return newFilesystemStore(sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, cfg.Options.Filesystem.Basedir)
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
