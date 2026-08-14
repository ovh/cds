package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// FilesystemStore implements ObjectStore interface with filesystem driver
type FilesystemStore struct {
	projectIntegration sdk.ProjectIntegration
	basedir            string
}

// newFilesystemStore creates a new ObjectStore with filesystem driver
func newFilesystemStore(ctx context.Context, projectIntegration sdk.ProjectIntegration, conf ConfigOptionsFilesystem) (*FilesystemStore, error) {
	log.Info(ctx, "ObjectStore> Initialize Filesystem driver on directory: %s", conf.Basedir)
	if conf.Basedir == "" {
		return nil, fmt.Errorf("artifact storage is filesystem, but --artifact-basedir is not provided")
	}

	fss := &FilesystemStore{projectIntegration: projectIntegration, basedir: conf.Basedir}
	return fss, nil
}

// TemporaryURLSupported returns true is temporary URL are supported
func (fss *FilesystemStore) TemporaryURLSupported() bool {
	return false
}

// GetProjectIntegration returns current projet Integration, nil otherwise
func (fss *FilesystemStore) GetProjectIntegration() sdk.ProjectIntegration {
	return fss.projectIntegration
}

// Status return filesystem storage status
func (fss *FilesystemStore) Status(ctx context.Context) sdk.MonitoringStatusLine {
	if _, err := os.Stat(fss.basedir); os.IsNotExist(err) {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Filesystem Storage KO (" + err.Error() + ")", Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Filesystem Storage", Status: sdk.MonitoringStatusOK}
}

// Store store a object on disk. The content is written in a temporary file then moved in place, so
// that a reader never gets a partially written object and an object being replaced stays readable
// for the whole write.
func (fss *FilesystemStore) Store(o Object, data io.ReadCloser) (string, error) {
	defer data.Close()

	dst := path.Join(fss.basedir, o.GetPath())
	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", err
	}
	distfile := path.Join(dst, o.GetName())

	f, err := os.CreateTemp(dst, "."+o.GetName()+".tmp")
	if err != nil {
		return "", err
	}
	tmpfile := f.Name()
	defer os.Remove(tmpfile) // no-op once the file has been renamed

	if _, err := io.Copy(f, data); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(tmpfile, distfile); err != nil {
		return "", err
	}

	return distfile, nil
}

// Fetch lookup on disk for data
func (fss *FilesystemStore) Fetch(ctx context.Context, o Object) (io.ReadCloser, error) {
	dst := path.Join(fss.basedir, o.GetPath(), o.GetName())
	f, err := os.Open(dst)
	if errors.Is(err, os.ErrNotExist) {
		return f, sdk.WithStack(sdk.ErrNotFound)
	}
	return f, err
}

// Delete deletes data from disk
func (fss *FilesystemStore) Delete(ctx context.Context, o Object) error {
	dst := path.Join(fss.basedir, o.GetPath(), o.GetName())
	return os.RemoveAll(dst)
}

// DeleteContainer deletes a directory from disk
func (fss *FilesystemStore) DeleteContainer(ctx context.Context, containerPath string) error {
	// check, just to be sure...
	if strings.TrimSpace(containerPath) != "" && containerPath != "/" {
		dst := path.Join(fss.basedir, containerPath)
		return os.RemoveAll(dst)
	}
	return nil
}
