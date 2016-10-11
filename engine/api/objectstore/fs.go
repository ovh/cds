package objectstore

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// FilesystemStore implements ObjectStore interface with filesystem driver
type FilesystemStore struct {
	basedir string
}

// NewFilesystemStore creates a new ObjectStore with filesystem driver
func NewFilesystemStore(basedir string) (*FilesystemStore, error) {
	if basedir == "" {
		return nil, fmt.Errorf("artifact storage is filesystem, but --artifact-basedir is not provided")
	}

	fss := &FilesystemStore{basedir: basedir}
	return fss, nil
}

//Status return filesystem storage status
func (fss *FilesystemStore) Status() string {
	if _, err := os.Stat(fss.basedir); os.IsNotExist(err) {
		return "Filesystem Storage KO (" + err.Error() + ")"
	}
	return "Filesystem Storage OK"
}

// StoreArtifact create a new file on disk with artifact data
func (fss *FilesystemStore) StoreArtifact(art sdk.Artifact, data io.ReadCloser) (string, error) {
	p := fss.path(art)
	log.Notice("FilesystemStore.Store> New artifact '%s' in %s\n", art.Name, p)

	dir, _ := filepath.Split(p)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}

	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err == io.EOF {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	_, err = io.Copy(f, data)
	if err != nil {
		return "", err
	}

	return p, nil
}

// FetchArtifact lookup on disk for artifact data
func (fss *FilesystemStore) FetchArtifact(art sdk.Artifact) (io.ReadCloser, error) {
	p := fss.path(art)

	f, err := os.OpenFile(p, os.O_RDONLY, 0700)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// DeleteArtifact remove artifact data from disk
func (fss *FilesystemStore) DeleteArtifact(art sdk.Artifact) error {
	return os.Remove(fss.path(art))
}

// StorePlugin store a plugin in disk
func (fss *FilesystemStore) StorePlugin(art sdk.ActionPlugin, data io.ReadCloser) (string, error) {

	dst := path.Join(fss.basedir, "plugin")
	src, err := os.Open(art.Path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", err
	}
	distfile := path.Join(dst, art.Name)
	f, err := os.OpenFile(distfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(f, src)
	return distfile, err
}

// FetchPlugin lookup on disk for plugin data
func (fss *FilesystemStore) FetchPlugin(art sdk.ActionPlugin) (io.ReadCloser, error) {
	dst := path.Join(fss.basedir, "plugin", art.Name)
	return os.Open(dst)
}

// DeletePlugin lookup on disk for plugin data
func (fss *FilesystemStore) DeletePlugin(art sdk.ActionPlugin) error {
	dst := path.Join(fss.basedir, "plugin", art.Name)
	return os.RemoveAll(dst)
}

func (fss *FilesystemStore) path(art sdk.Artifact) string {
	dir := fmt.Sprintf("%s/%s/%s/%s", art.Project, art.Application, art.Environment, art.Pipeline)
	return path.Join(fss.basedir, dir, art.Tag, art.Name)
}
