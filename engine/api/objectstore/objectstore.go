package objectstore

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

var storage Driver

//Status is for status handler
func Status() string {
	if storage == nil {
		return "KO : Store not initialized"
	}

	return storage.Status()
}

//StoreArtifact an artifact with default objectstore driver
func StoreArtifact(art sdk.Artifact, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.Store(&art, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchArtifact an artifact with default objectstore driver
func FetchArtifact(art sdk.Artifact) (io.ReadCloser, error) {
	if storage != nil {
		return storage.Fetch(&art)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeleteArtifact an artifact with default objectstore driver
func DeleteArtifact(art sdk.Artifact) error {
	if storage != nil {
		return storage.Delete(&art)
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
		return NewOpenstackStore(c, cfg.Options.Openstack.Address,
			cfg.Options.Openstack.Username,
			cfg.Options.Openstack.Password,
			cfg.Options.Openstack.Tenant,
			cfg.Options.Openstack.Region,
			cfg.Options.Openstack.ContainerPrefix)
	case Filesystem:
		return NewFilesystemStore(cfg.Options.Filesystem.Basedir)
	default:
		return nil, fmt.Errorf("Invalid flag --artifact-mode")
	}
}

//StreamFile streams file
func StreamFile(w io.Writer, f io.ReadCloser) error {
	n, err := copyBuffer(w, f, nil)
	if err != nil {
		return fmt.Errorf("cannot stream to client [%dbytes copied]: %s", n, err)
	}
	return nil
}

func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		buf = make([]byte, 32*1024)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = fmt.Errorf("writer: %s", ew)
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = fmt.Errorf("reader: %s", er)
			break
		}
	}
	return written, err
}
