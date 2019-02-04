package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
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

// Init initialise a new ArtifactStorage
func Init(c context.Context, cfg Config) (Driver, error) {
	switch cfg.Kind {
	case Openstack, Swift:
		return newSwiftStore(cfg.Options.Openstack.Address,
			cfg.Options.Openstack.Username,
			cfg.Options.Openstack.Password,
			cfg.Options.Openstack.Region,
			cfg.Options.Openstack.Tenant,
			cfg.Options.Openstack.Domain,
			cfg.Options.Openstack.ContainerPrefix,
			cfg.Options.Openstack.DisableTempURL)
	case Filesystem:
		return newFilesystemStore(cfg.Options.Filesystem.Basedir)
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
