package objectstore

import (
	"fmt"
	"io"
	"time"

	"github.com/ncw/swift"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SwiftStore implements ObjectStore interface with openstack swift implementation
type SwiftStore struct {
	swift.Connection
	containerPrefix    string
	disableTempURL     bool
	projectIntegration sdk.ProjectIntegration
}

var swiftServeStaticFileEnabled bool

func newSwiftStore(integration sdk.ProjectIntegration, conf ConfigOptionsOpenstack) (*SwiftStore, error) {
	log.Info("ObjectStore> Initialize Swift driver on url: %s", conf.Address)
	s := &SwiftStore{
		Connection: swift.Connection{
			AuthUrl:  conf.Address,
			Region:   conf.Region,
			Tenant:   conf.Tenant,
			Domain:   conf.Domain,
			UserName: conf.Username,
			ApiKey:   conf.Password,
		},
		containerPrefix:    conf.ContainerPrefix,
		disableTempURL:     conf.DisableTempURL,
		projectIntegration: integration,
	}
	if err := s.Authenticate(); err != nil {
		return nil, sdk.WrapError(err, "Unable to authenticate on swift storage")
	}
	return s, nil
}

// TemporaryURLSupported returns true is temporary URL are supported
func (s *SwiftStore) TemporaryURLSupported() bool {
	return !s.disableTempURL
}

// GetProjectIntegration returns current projet Integration, nil otherwise
func (s *SwiftStore) GetProjectIntegration() sdk.ProjectIntegration {
	return s.projectIntegration
}

// Status returns the status of swift account
func (s *SwiftStore) Status() sdk.MonitoringStatusLine {
	info, _, err := s.Account()
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Swift KO" + err.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{
		Component: "Object-Store",
		Value:     fmt.Sprintf("Swift OK (%d containers, %d objects, %d bytes used", info.Containers, info.Containers, info.BytesUsed),
		Status:    sdk.MonitoringStatusOK,
	}
}

// ServeStaticFiles send files to serve static files with the entrypoint of html page and return public URL taking a tar file
func (s *SwiftStore) ServeStaticFiles(o Object, entrypoint string, data io.ReadCloser) (string, error) {
	if !swiftServeStaticFileEnabled {
		return "", sdk.WithStack(sdk.ErrNotImplemented)
	}

	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	log.Debug("SwiftStore> Storing /%s/%s\n", container, object)

	if entrypoint == "" {
		entrypoint = "index.html"
	}
	headers := map[string]string{
		"X-Web-Mode":                    "TRUE",
		"X-Container-Meta-Web-Listings": "TRUE",
		"X-Container-Meta-Web-Index":    entrypoint,
		"X-Container-Read":              ".r:*,.rlistings",
		"X-Delete-After":                fmt.Sprintf("%d", time.Now().Add(time.Hour*1500).Unix()), //TODO: to delete when purge will be developed
	}

	log.Debug("SwiftStore> creating container %s", container)
	if err := s.ContainerCreate(container, headers); err != nil {
		return "", sdk.WrapError(err, "Unable to create container %s", container)
	}

	log.Debug("SwiftStore> creating object %s/%s", container, object)
	res, errU := s.BulkUpload(container, data, "tar", nil)
	if errU != nil {
		return "", sdk.WrapError(errU, "SwiftStore> Unable to bulk upload %s : %v : %+v", object, errU, res.Errors)
	}

	if err := data.Close(); err != nil {
		return "", sdk.WrapError(err, "Unable to close data buffer")
	}

	return s.StorageUrl + "/" + container, nil

}

// Store stores in swift
func (s *SwiftStore) Store(o Object, data io.ReadCloser) (string, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	log.Debug("SwiftStore> Storing /%s/%s\n", container, object)
	log.Debug("SwiftStore> creating container %s", container)
	if err := s.ContainerCreate(container, nil); err != nil {
		return "", sdk.WrapError(err, "Unable to create container %s", container)
	}

	log.Debug("SwiftStore> creating object %s/%s", container, object)

	file, errC := s.ObjectCreate(container, object, false, "", "application/octet-stream", nil)
	if errC != nil {
		return "", sdk.WrapError(errC, "SwiftStore> Unable to create object %s", object)
	}

	log.Debug("SwiftStore> copy object %s/%s", container, object)
	if _, err := io.Copy(file, data); err != nil {
		_ = file.Close()
		_ = data.Close()
		return "", sdk.WrapError(err, "Unable to copy object buffer %s", object)
	}

	if err := file.Close(); err != nil {
		return "", sdk.WrapError(err, "Unable to close object buffer %s", object)
	}

	if err := data.Close(); err != nil {
		return "", sdk.WrapError(err, "Unable to close data buffer")
	}

	return container + "/" + object, nil
}

// Fetch an object from swift
func (s *SwiftStore) Fetch(o Object) (io.ReadCloser, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	pipeReader, pipeWriter := io.Pipe()
	log.Debug("SwiftStore> Fetching /%s/%s\n", container, object)

	go func() {
		log.Debug("SwiftStore> downloading object %s/%s", container, object)

		if _, err := s.ObjectGet(container, object, pipeWriter, false, nil); err != nil {
			log.Error("SwiftStore> Unable to get object %s/%s: %s", container, object, err)
		}

		log.Debug("SwiftStore> object %s%s downloaded", container, object)
		pipeWriter.Close()
	}()
	return pipeReader, nil
}

// Delete an object from swift
func (s *SwiftStore) Delete(o Object) error {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	if err := s.ObjectDelete(container, object); err != nil {
		if err.Error() == "Object Not Found" {
			log.Info("Delete.SwiftStore: %s/%s: %s", container, object, err)
			return nil
		}
		return sdk.WrapError(err, "Unable to delete object")
	}
	return nil
}

// StoreURL returns a temporary url and a secret key to store an object
func (s *SwiftStore) StoreURL(o Object, contentType string) (string, string, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	if err := s.ContainerCreate(container, nil); err != nil {
		return "", "", sdk.WrapError(err, "Unable to create container %s", container)
	}

	key, err := s.containerKey(container)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to get container key %s", container)
	}

	url := s.ObjectTempUrl(container, object, string(key), "PUT", time.Now().Add(time.Hour))
	return url, string(key), nil
}

// ServeStaticFilesURL returns a temporary url and a secret key to serve static files in a container
func (s *SwiftStore) ServeStaticFilesURL(o Object, entrypoint string) (string, string, error) {
	if !swiftServeStaticFileEnabled {
		return "", "", sdk.WithStack(sdk.ErrNotImplemented)
	}
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	if entrypoint == "" {
		entrypoint = "index.html"
	}

	headers := map[string]string{
		"X-Web-Mode":                    "TRUE",
		"X-Container-Meta-Web-Listings": "TRUE",
		"X-Container-Meta-Web-Index":    entrypoint,
		"X-Container-Read":              ".r:*,.rlistings",
	}
	if err := s.ContainerCreate(container, headers); err != nil {
		return "", "", sdk.WrapError(err, "Unable to create container %s", container)
	}

	key, err := s.containerKey(container)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to get container key %s", container)
	}

	url := s.ObjectTempUrl(container, object, string(key), "PUT", time.Now().Add(time.Hour))
	return url, string(key), nil

}

func (s *SwiftStore) containerKey(container string) (string, error) {
	_, headers, err := s.Container(container)
	if err != nil {
		return "", sdk.WrapError(err, "Unable to get container %s", container)
	}

	key := headers["X-Container-Meta-Temp-Url-Key"]
	if key == "" {
		log.Debug("SwiftStore> Creating new session key for %s", container)
		key = sdk.UUID()

		log.Debug("SwiftStore> Update container %s metadata", container)
		if err := s.ContainerUpdate(container, swift.Headers{"X-Container-Meta-Temp-Url-Key": key}); err != nil {
			return "", sdk.WrapError(err, "Unable to update container metadata %s", container)
		}
	}

	return key, nil
}

// FetchURL returns a temporary url and a secret key to fetch an object
func (s *SwiftStore) FetchURL(o Object) (string, string, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	key, err := s.containerKey(container)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to get container key %s", container)
	}

	url := s.ObjectTempUrl(container, object, string(key), "GET", time.Now().Add(time.Hour))

	log.Debug("SwiftStore> Fetch URL: %s", string(url))
	return url + "&extract-archive=tar.gz", string(key), nil
}
