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
func Status() sdk.MonitoringStatusLine {
	if storage == nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Store not initialized", Status: sdk.MonitoringStatusAlert}
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

//Store an object with default objectstore driver
func Store(o Object, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.Store(o, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//Fetch an object with default objectstore driver
func Fetch(o Object) (io.ReadCloser, error) {
	if storage != nil {
		return storage.Fetch(o)
	}
	return nil, fmt.Errorf("store not initialized")
}

//Delete an object with default objectstore driver
func Delete(o Object) error {
	if storage != nil {
		return storage.Delete(o)
	}
	return fmt.Errorf("store not initialized")
}

//FetchTempURL returns a temp URL
func FetchTempURL(o Object) (string, error) {
	if storage == nil {
		return "", fmt.Errorf("store not initialized")
	}

	s, ok := storage.(DriverWithRedirect)
	if !ok {
		return "", fmt.Errorf("temp URL not supported")
	}

	url, _, err := s.FetchURL(o)
	return url, err
}

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack / Swift
// - Filesystem
type Driver interface {
	Status() sdk.MonitoringStatusLine
	Store(o Object, data io.ReadCloser) (string, error)
	Fetch(o Object) (io.ReadCloser, error)
	Delete(o Object) error
}

// DriverWithRedirect has to be implemented if your storage backend supports temp url
type DriverWithRedirect interface {
	// StoreURL returns a temporary url and a secret key to store an object
	StoreURL(o Object) (url string, key string, err error)
	// FetchURL returns a temporary url and a secret key to fetch an object
	FetchURL(o Object) (url string, key string, err error)
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
