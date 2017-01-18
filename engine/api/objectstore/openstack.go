package objectstore

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/ovh/cds/engine/log"
)

// OpenstackStore implements ObjectStore interface with openstack implementation
type OpenstackStore struct {
	address  string
	user     string
	password string
	tenant   string
	region   string
	token    *Token
	endpoint string
}

// NewOpenstackStore create a new ObjectStore with openstack driver and check configuration
func NewOpenstackStore(address, user, password, tenant, region string) (*OpenstackStore, error) {
	log.Notice("Objectstore> Initialize Swift(Openstack) driver on address: %s, tenant: %s, region: %s", address, tenant, region)
	if address == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_address is not provided")
	}

	if user == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_user is not provided")
	}

	if password == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_password is not provided")
	}

	if tenant == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_tenant is not provided")
	}

	ops := &OpenstackStore{
		address:  address,
		user:     user,
		password: password,
		tenant:   tenant,
		region:   region,
	}

	var err error
	ops.token, ops.endpoint, err = getToken(user, password, address, tenant, region)
	if err != nil {
		return nil, err
	}
	go ops.refreshTokenRoutine()

	log.Debug("NewOpenstackStore> Got token %dchar at %s\n", len(ops.token.ID), ops.endpoint)
	return ops, nil
}

//Status return Openstack storage status
func (ops *OpenstackStore) Status() string {
	if err := account(ops.token.ID, ops.endpoint); err != nil {
		return "Openstack KO (" + err.Error() + ")"
	}
	return "Openstack OK"
}

// Delete should delete on openstack
func (ops *OpenstackStore) Delete(o Object) error {
	return deleteObject(ops.token.ID, ops.endpoint, o.GetPath(), o.GetName())
}

// Store stores in openstack
func (ops *OpenstackStore) Store(o Object, data io.ReadCloser) (string, error) {
	container := o.GetPath()
	object := o.GetName()

	ops.escape(container, object)

	log.Info("OpenstackStore> Storing /%s/%s\n", container, object)

	// Create container if it doesn't exist
	err := createContainer(ops.token.ID, ops.endpoint, container)
	if err != nil {
		log.Warning("OpenstackStore.Store> Cannot create container: %s\n", err)
		return "", err
	}

	// Create object
	err = createObject(ops.token.ID, ops.endpoint, container, object, data)
	if err != nil {
		log.Warning("OpenstackStore.Store> Cannot create object: %s\n", err)
		return "", err
	}

	return container + "/" + object, nil
}

// Fetch lookup on openstack to fetch data
func (ops *OpenstackStore) Fetch(o Object) (io.ReadCloser, error) {
	container := o.GetPath()
	object := o.GetName()
	ops.escape(container, object)

	log.Info("OpenstackStore> Fetching /%s/%s\n", container, object)

	data, err := fetchObject(ops.token.ID, ops.endpoint, container, object)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (ops *OpenstackStore) escape(container, object string) (string, string) {
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	object = url.QueryEscape(object)
	object = strings.Replace(object, "/", "-", -1)
	return container, object
}
