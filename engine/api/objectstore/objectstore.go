package objectstore

import (
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
		return storage.StoreArtifact(art, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchArtifact an artifact with default objectstore driver
func FetchArtifact(art sdk.Artifact) (io.ReadCloser, error) {
	if storage != nil {
		return storage.FetchArtifact(art)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeleteArtifact an artifact with default objectstore driver
func DeleteArtifact(art sdk.Artifact) error {
	if storage != nil {
		return storage.DeleteArtifact(art)
	}
	return fmt.Errorf("store not initialized")
}

//StorePlugin call StorePlugin on the common driver
func StorePlugin(art sdk.ActionPlugin, data io.ReadCloser) (string, error) {
	if storage != nil {
		return storage.StorePlugin(art, data)
	}
	return "", fmt.Errorf("store not initialized")
}

//FetchPlugin call FetchPlugin on the common driver
func FetchPlugin(art sdk.ActionPlugin) (io.ReadCloser, error) {
	if storage != nil {
		return storage.FetchPlugin(art)
	}
	return nil, fmt.Errorf("store not initialized")
}

//DeletePlugin call DeletePlugin on the common driver
func DeletePlugin(art sdk.ActionPlugin) error {
	if storage != nil {
		return storage.DeletePlugin(art)
	}
	return fmt.Errorf("store not initialized")
}

// Driver allows artifact to be stored and retrieve the same way to any backend
// - Openstack ObjectStore
// - Filesystem
type Driver interface {
	Status() string
	StoreArtifact(art sdk.Artifact, data io.ReadCloser) (string, error)
	FetchArtifact(art sdk.Artifact) (io.ReadCloser, error)
	DeleteArtifact(art sdk.Artifact) error
	StorePlugin(art sdk.ActionPlugin, data io.ReadCloser) (string, error)
	FetchPlugin(art sdk.ActionPlugin) (io.ReadCloser, error)
	DeletePlugin(art sdk.ActionPlugin) error
}

// Initialize setup wanted ObjectStore driver
func Initialize(mode, address, user, password, basedir string) error {
	var err error
	storage, err = New(mode, address, user, password, basedir)
	if err != nil {
		return err
	}

	return nil
}

// New initialise a new ArtifactStorage
func New(mode, address, user, password, basedir string) (Driver, error) {
	switch mode {
	case "openstack":
		return NewOpenstackStore(address, user, password)
	case "filesystem":
		return NewFilesystemStore(basedir)
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
