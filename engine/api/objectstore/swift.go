package objectstore

import (
	"fmt"
	"io"
	"time"

	"github.com/ncw/swift"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SwiftStore implements ObjectStore interface with openstack swift implementation
type SwiftStore struct {
	swift.Connection
	containerprefix string
}

// NewSwiftStore create a new ObjectStore with openstack driver and check configuration
func NewSwiftStore(authURL, user, password, region, tenant, domain, containerprefix string) (Driver, error) {
	s := SwiftStore{
		swift.Connection{
			AuthUrl:  authURL,
			Region:   region,
			Tenant:   tenant,
			Domain:   domain,
			UserName: user,
			ApiKey:   password,
		}, containerprefix}

	if err := s.Authenticate(); err != nil {
		return nil, sdk.WrapError(err, "Swift> Unable to authenticate")
	}
	return &s, nil
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

// Store stores in swift
func (s *SwiftStore) Store(o Object, data io.ReadCloser) (string, error) {
	container := s.containerprefix + o.GetPath()
	object := o.GetName()
	escape(container, object)
	log.Debug("SwiftStore> Storing /%s/%s\n", container, object)

	log.Debug("SwiftStore> creating container %s", container)
	if err := s.ContainerCreate(container, nil); err != nil {
		return "", sdk.WrapError(err, "SwiftStore> Unable to create container %s", container)
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
		return "", sdk.WrapError(err, "SwiftStore> Unable to copy object buffer %s", object)
	}

	if err := file.Close(); err != nil {
		return "", sdk.WrapError(err, "SwiftStore> Unable to close object buffer %s", object)
	}

	if err := data.Close(); err != nil {
		return "", sdk.WrapError(err, "SwiftStore> Unable to close data buffer")
	}

	return container + "/" + object, nil
}

// Fetch an object from swift
func (s *SwiftStore) Fetch(o Object) (io.ReadCloser, error) {
	container := s.containerprefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	pipeReader, pipeWriter := io.Pipe()
	log.Debug("OpenstacSwiftStorekStore> Fetching /%s/%s\n", container, object)

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
	container := s.containerprefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	if err := s.ObjectDelete(container, object); err != nil {
		if err.Error() == "Object Not Found" {
			log.Info("Delete.SwiftStore: %s/%s: %s", container, object, err)
			return nil
		}
		return sdk.WrapError(err, "SwiftStore> Unable to delete object")
	}
	return nil
}

// StoreURL returns a temporary url and a secret key to store an object
func (s *SwiftStore) StoreURL(o Object) (string, string, error) {
	container := s.containerprefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	if err := s.ContainerCreate(container, nil); err != nil {
		return "", "", sdk.WrapError(err, "SwiftStore> Unable to create container %s", container)
	}

	key, err := s.containerKey(container)
	if err != nil {
		return "", "", sdk.WrapError(err, "SwiftStore> Unable to get container key %s", container)
	}

	url := s.ObjectTempUrl(container, object, string(key), "PUT", time.Now().Add(time.Hour))
	return url, string(key), nil
}

func (s *SwiftStore) containerKey(container string) (string, error) {
	_, headers, err := s.Container(container)
	if err != nil {
		return "", sdk.WrapError(err, "SwiftStore> Unable to get container %s", container)
	}

	key := headers["X-Container-Meta-Temp-Url-Key"]
	if key == "" {
		log.Debug("SwiftStore> Creating new session key for %s", container)
		skey, _ := sessionstore.NewSessionKey()
		key = string(skey)

		log.Debug("SwiftStore> Update container %s metadata", container)
		if err := s.ContainerUpdate(container, swift.Headers{"X-Container-Meta-Temp-Url-Key": key}); err != nil {
			return "", sdk.WrapError(err, "SwiftStore> Unable to update container metadata %s", container)
		}
	}

	return key, nil
}

// FetchURL returns a temporary url and a secret key to fetch an object
func (s *SwiftStore) FetchURL(o Object) (string, string, error) {
	container := s.containerprefix + o.GetPath()
	object := o.GetName()
	escape(container, object)

	key, err := s.containerKey(container)
	if err != nil {
		return "", "", sdk.WrapError(err, "SwiftStore> Unable to get container key %s", container)
	}

	url := s.ObjectTempUrl(container, object, string(key), "GET", time.Now().Add(time.Hour))

	log.Debug("SwiftStore> Fetch URL: %s", string(url))
	return url, string(key), nil
}
