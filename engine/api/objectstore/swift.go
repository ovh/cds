package objectstore

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ncw/swift"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// SwiftStore implements ObjectStore interface with openstack swift implementation
type SwiftStore struct {
	swift.Connection
	containerPrefix    string
	disableTempURL     bool
	projectIntegration sdk.ProjectIntegration
}

var swiftServeStaticFileEnabled bool

func newSwiftStore(ctx context.Context, integration sdk.ProjectIntegration, conf ConfigOptionsOpenstack) (*SwiftStore, error) {
	log.Info(ctx, "ObjectStore> Initialize Swift driver on url: %s", conf.Address)
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
func (s *SwiftStore) Status(ctx context.Context) sdk.MonitoringStatusLine {
	info, _, err := s.Account()
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "Object-Store", Value: "Swift KO" + err.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{
		Component: "Object-Store",
		Value:     fmt.Sprintf("Swift OK (%d containers, %d objects, %d bytes used", info.Containers, info.Objects, info.BytesUsed),
		Status:    sdk.MonitoringStatusOK,
	}
}

// Store stores in swift
func (s *SwiftStore) Store(o Object, data io.ReadCloser) (string, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	log.Debug(context.Background(), "SwiftStore> Storing /%s/%s\n", container, object)
	log.Debug(context.Background(), "SwiftStore> creating container %s", container)
	if err := s.ContainerCreate(container, nil); err != nil {
		return "", sdk.WrapError(err, "Unable to create container %s", container)
	}

	log.Debug(context.Background(), "SwiftStore> creating object %s/%s", container, object)

	file, errC := s.ObjectCreate(container, object, false, "", "application/octet-stream", nil)
	if errC != nil {
		return "", sdk.WrapError(errC, "SwiftStore> Unable to create object %s", object)
	}

	log.Debug(context.Background(), "SwiftStore> copy object %s/%s", container, object)
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
func (s *SwiftStore) Fetch(ctx context.Context, o Object) (io.ReadCloser, error) {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	pipeReader, pipeWriter := io.Pipe()
	log.Debug(ctx, "SwiftStore> Fetching /%s/%s\n", container, object)

	go func() {
		log.Debug(ctx, "SwiftStore> downloading object %s/%s", container, object)

		if _, err := s.ObjectGet(container, object, pipeWriter, false, nil); err != nil {
			log.Error(ctx, "SwiftStore> Unable to get object %s/%s: %s", container, object, err)
		}

		log.Debug(ctx, "SwiftStore> object %s%s downloaded", container, object)
		pipeWriter.Close()
	}()
	return pipeReader, nil
}

// Delete deletes an object from swift
func (s *SwiftStore) Delete(ctx context.Context, o Object) error {
	container := s.containerPrefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	if err := s.ObjectDelete(container, object); err != nil {
		if err.Error() == swift.ObjectNotFound.Text {
			log.Info(ctx, "Delete.SwiftStore: %s/%s: %s", container, object, err)
			return nil
		}
		return sdk.WrapError(err, "Unable to delete object")
	}
	return nil
}

// DeleteContainer deletes a container from swift
func (s *SwiftStore) DeleteContainer(ctx context.Context, containerPath string) error {
	container := s.containerPrefix + containerPath
	escape(container, "")

	if err := s.ContainerDelete(container); err != nil {
		if err.Error() == swift.ContainerNotFound.Text {
			log.Info(ctx, "Delete.SwiftStore: %s: %s", container, err)
			return nil
		}
		return sdk.WrapError(err, "Unable to delete container")
	}
	log.Debug(ctx, "Delete.SwiftStore: %s is deleted", container)
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

func (s *SwiftStore) containerKey(container string) (string, error) {
	_, headers, err := s.Container(container)
	if err != nil {
		return "", sdk.WrapError(err, "Unable to get container %s", container)
	}

	key := headers["X-Container-Meta-Temp-Url-Key"]
	if key == "" {
		log.Debug(context.Background(), "SwiftStore> Creating new session key for %s", container)
		key = sdk.UUID()

		log.Debug(context.Background(), "SwiftStore> Update container %s metadata", container)
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

	log.Debug(context.Background(), "SwiftStore> Fetch URL: %s", string(url))
	return url + "&extract-archive=tar.gz", string(key), nil
}
