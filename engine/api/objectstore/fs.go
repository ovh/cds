package objectstore

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// FilesystemStore implements ObjectStore interface with filesystem driver
type FilesystemStore struct {
	basedir string
}

// NewFilesystemStore creates a new ObjectStore with filesystem driver
func NewFilesystemStore(basedir string) (*FilesystemStore, error) {
	log.Info("Objectstore> Initialize Filesystem driver on directory: %s", basedir)
	if basedir == "" {
		return nil, fmt.Errorf("artifact storage is filesystem, but --artifact-basedir is not provided")
	}

	fss := &FilesystemStore{basedir: basedir}
	return fss, nil
}

//Status return filesystem storage status
func (fss *FilesystemStore) Status() sdk.MonitoringStatusLine {
	if _, err := os.Stat(fss.basedir); os.IsNotExist(err) {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Filesystem Storage KO (" + err.Error() + ")", Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Filesystem Storage", Status: sdk.MonitoringStatusOK}
}

// Store store a object on disk
func (fss *FilesystemStore) Store(o Object, data io.ReadCloser) (string, error) {
	dst := path.Join(fss.basedir, o.GetPath())
	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", err
	}
	distfile := path.Join(dst, o.GetName())
	f, err := os.OpenFile(distfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(f, data)
	defer data.Close()
	return distfile, err
}

// Fetch lookup on disk for data
func (fss *FilesystemStore) Fetch(o Object) (io.ReadCloser, error) {
	dst := path.Join(fss.basedir, o.GetPath(), o.GetName())
	return os.Open(dst)
}

// Delete data on disk
func (fss *FilesystemStore) Delete(o Object) error {
	dst := path.Join(fss.basedir, o.GetPath(), o.GetName())
	return os.RemoveAll(dst)
}
