package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

var storage Driver
var instance sdk.ArtifactsStore

//Status is for status handler
func Status() string {
	if storage == nil {
		return "KO : Store not initialized"
	}

	return storage.Status()
}

// Instance returns the objectstore singleton
func Instance() sdk.ArtifactsStore {
	return instance
}

// Storage returns the Driver singleton
func Storage() Driver {
	return storage
}

//StoreArtifact an artifact with default objectstore driver
func StoreArtifact(o Object, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.Store(o, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchArtifact an artifact with default objectstore driver
func FetchArtifact(o Object) (io.ReadCloser, error) {
	if storage != nil {
		return storage.Fetch(o)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeleteArtifact an artifact with default objectstore driver
func DeleteArtifact(o Object) error {
	if storage != nil {
		return storage.Delete(o)
	}
	return fmt.Errorf("store not initialized")
}

//StorePlugin call Store on the common driver
func StorePlugin(art sdk.ActionPlugin, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.Store(&art, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchPlugin call Fetch on the common driver
func FetchPlugin(art sdk.ActionPlugin) (io.ReadCloser, error) {
	if storage != nil {
		return storage.Fetch(&art)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeletePlugin call Delete on the common driver
func DeletePlugin(art sdk.ActionPlugin) error {
	if storage != nil {
		return storage.Delete(&art)
	}
	return fmt.Errorf("store not initialized")
}

//StoreTemplateExtension call Store on the common driver
func StoreTemplateExtension(tmpl sdk.TemplateExtension, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.Store(&tmpl, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchTemplateExtension call Fetch on the common driver
func FetchTemplateExtension(tmpl sdk.TemplateExtension) (io.ReadCloser, error) {
	if storage != nil {
		return storage.Fetch(&tmpl)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeleteTemplateExtension call Delete on the common driver
func DeleteTemplateExtension(tmpl sdk.TemplateExtension) error {
	if storage != nil {
		return storage.Delete(&tmpl)
	}
	return fmt.Errorf("store not initialized")
}

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
	Status() string
	Store(o Object, data io.ReadCloser) (string, error)
	Fetch(o Object) (io.ReadCloser, error)
	Delete(o Object) error
}

// DriverWithRedirect has to be implemented if your storage backend supports temp url
type DriverWithRedirect interface {
	StoreURL(o Object) (string, string, error)
	FetchURL(o Object) (string, string, error)
}

// Initialize setup wanted ObjectStore driver
func Initialize(c context.Context, cfg Config) error {
	var err error
	storage, err = New(c, cfg)
	if err != nil {
		return err
	}
	return nil
}

// Kind will define const defining all supported objecstore drivers
type Kind int

// These are the defined objecstore drivers
const (
	Openstack Kind = iota
	Filesystem
	Swift
)

//TODO Use github.com/graymeta/stow

// Config represents all the configuration for all objecstore drivers
type Config struct {
	Kind    Kind
	Options ConfigOptions
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
	Region          string
	ContainerPrefix string
}

// ConfigOptionsFilesystem is used by ConfigOptions
type ConfigOptionsFilesystem struct {
	Basedir string
}

// New initialise a new ArtifactStorage
func New(c context.Context, cfg Config) (Driver, error) {
	switch cfg.Kind {
	case Openstack, Swift:
		instance = sdk.ArtifactsStore{
			Name:                  "Swift",
			Private:               false,
			TemporaryURLSupported: true,
		}
		return NewSwiftStore(cfg.Options.Openstack.Address,
			cfg.Options.Openstack.Username,
			cfg.Options.Openstack.Password,
			cfg.Options.Openstack.Region,
			cfg.Options.Openstack.Tenant,
			cfg.Options.Openstack.ContainerPrefix)
	case Filesystem:
		instance = sdk.ArtifactsStore{
			Name:                  "Local FS",
			Private:               false,
			TemporaryURLSupported: false,
		}
		return NewFilesystemStore(cfg.Options.Filesystem.Basedir)
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
