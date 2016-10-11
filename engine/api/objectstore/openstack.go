package objectstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// OpenstackStore implements ObjectStore interface with openstack implementation
type OpenstackStore struct {
	address  string
	user     string
	password string
	token    *Token
	endpoint string
}

// NewOpenstackStore create a new ObjectStore with openstack driver and check configuration
func NewOpenstackStore(address, user, password string) (*OpenstackStore, error) {

	if address == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_address is not provided")
	}

	if user == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_user is not provided")
	}

	if password == "" {
		return nil, fmt.Errorf("artifact storage is openstack, but flag --artifact_password is not provided")
	}

	ops := &OpenstackStore{
		address:  address,
		user:     user,
		password: password,
	}

	var err error
	ops.token, ops.endpoint, err = getToken(user, password, address, "T_cds")
	if err != nil {
		return nil, err
	}
	go ops.refreshTokenRoutine()

	log.Notice("NewOpenstackStore> Got token %dchar at %s\n", len(ops.token.ID), ops.endpoint)
	return ops, nil
}

func (ops *OpenstackStore) refreshTokenRoutine() {

	for {
		time.Sleep(20 * time.Hour)

		tk, endpoint, err := getToken(ops.user, ops.password, ops.address, "T_cds")
		if err != nil {
			log.Critical("refreshTokenRoutine> Cannot refresh token: %s\n", err)
			continue
		}
		ops.token = tk
		ops.endpoint = endpoint
	}
}

// StoreArtifact creates a new object in openstack with artifact data
func (ops *OpenstackStore) StoreArtifact(art sdk.Artifact, data io.ReadCloser) (string, error) {
	container, object := ops.format(art.Name, art.Project, art.Application, art.Environment, art.Pipeline, art.Tag)
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

// FetchArtifact retrieves artifact data from openstack
func (ops *OpenstackStore) FetchArtifact(art sdk.Artifact) (io.ReadCloser, error) {
	container, object := ops.format(art.Name, art.Project, art.Application, art.Environment, art.Pipeline, art.Tag)
	log.Info("OpenstackStore> Fetching /%s/%s\n", container, object)

	data, err := fetchObject(ops.token.ID, ops.endpoint, container, object)
	if err != nil {
		return nil, err
	}

	return data, nil
}

//Status return Openstack storage status
func (ops *OpenstackStore) Status() string {
	if err := account(ops.token.ID, ops.endpoint); err != nil {
		return "Openstack KO (" + err.Error() + ")"
	}
	return "Openstack OK"
}

// DeleteArtifact removes artifact data from openstack
func (ops *OpenstackStore) DeleteArtifact(art sdk.Artifact) error {
	container, object := ops.format(art.Name, art.Project, art.Application, art.Environment, art.Pipeline, art.Tag)
	log.Info("OpenstackStore> Deleting /%s/%s\n", container, object)

	return deleteObject(ops.token.ID, ops.endpoint, container, object)
}

// StorePlugin store a plugin in openstack
func (ops *OpenstackStore) StorePlugin(art sdk.ActionPlugin, data io.ReadCloser) (string, error) {
	container, object := ops.format(art.Name, "plugins")
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

// FetchPlugin lookup on disk for plugin data
func (ops *OpenstackStore) FetchPlugin(art sdk.ActionPlugin) (io.ReadCloser, error) {
	container, object := ops.format(art.Name, "plugins")
	log.Info("OpenstackStore> Fetching /%s/%s\n", container, object)

	data, err := fetchObject(ops.token.ID, ops.endpoint, container, object)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DeletePlugin lookup on disk for plugin data
func (ops *OpenstackStore) DeletePlugin(art sdk.ActionPlugin) error {
	return nil
}

func (ops *OpenstackStore) format(x string, y ...string) (container string, object string) {
	container = strings.Join(y, "-")

	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)

	object = url.QueryEscape(x)
	object = strings.Replace(object, "/", "-", -1)
	return
}

//////////// OPENSTACK HANDLERS //////////

type auth struct {
	Auth struct {
		Tenant string `json:"tenantName"`
		Creds  struct {
			User     string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
	} `json:"auth"`
}

// AccessType describe the access given by token
type AccessType struct {
	Token          Token                 `json:"token"`
	User           interface{}           `json:"id"`
	ServiceCatalog []ServiceCatalogEntry `json:"servicecatalog"`
}

// AuthToken is a specific openstack format
type AuthToken struct {
	Access AccessType `json:"access"`
}

// Token represent an openstack token
type Token struct {
	ID      string    `json:"id"`
	Expires time.Time `json:"expires"`
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

// ServiceCatalogEntry is an openstack specific object
type ServiceCatalogEntry struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Endpoints []ServiceEndpoint `json:"endpoints"`
}

// ServiceEndpoint describe an openstack endpoint
type ServiceEndpoint struct {
	Type        string `json:"type"`
	Region      string `json:"region"`
	PublicURL   string `json:"publicurl"`
	AdminURL    string `json:"adminurl"`
	InternalURL string `json:"internalurl"`
	VersionID   string `json:"versionid"`
}

func account(token string, url string) error {
	uri := fmt.Sprintf("%s", url)
	req, err := http.NewRequest("HEAD", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return unmarshalOpenstackError(rbody, resp.Status)
	}

	return nil
}

func deleteObject(token string, url string, account string, objectname string) error {
	uri := fmt.Sprintf("%s/%s/%s", url, account, objectname)
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return unmarshalOpenstackError(rbody, resp.Status)
	}

	return nil
}

func fetchObject(token string, url string, account string, objectname string) (io.ReadCloser, error) {
	uri := fmt.Sprintf("%s/%s/%s", url, account, objectname)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read body")
		}
		return nil, unmarshalOpenstackError(rbody, resp.Status)
	}

	return resp.Body, nil
}

func createObject(token string, url string, account string, objectname string, data io.ReadCloser) error {
	uri := fmt.Sprintf("%s/%s/%s", url, account, objectname)
	req, err := http.NewRequest("PUT", uri, data)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return fmt.Errorf("%s (%s)", uri, unmarshalOpenstackError(rbody, resp.Status))
	}

	return nil
}

func createContainer(token string, url string, account string) error {
	uri := fmt.Sprintf("%s/%s", url, account)
	req, err := http.NewRequest("PUT", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read body")
		}
		return fmt.Errorf("%s (%s)", uri, unmarshalOpenstackError(rbody, resp.Status))
	}

	return nil
}

func getToken(user string, password string, url string, project string) (*Token, string, error) {
	var endpoint string

	a := auth{}
	a.Auth.Tenant = project
	a.Auth.Creds.User = user
	a.Auth.Creds.Password = password

	data, err := json.Marshal(a)
	if err != nil {
		return nil, endpoint, err
	}

	uri := fmt.Sprintf("%s/v2.0/tokens", url)
	req, err := http.NewRequest("POST", uri, bytes.NewReader(data))
	if err != nil {
		return nil, endpoint, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, endpoint, err
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "json") != true {
		return nil, endpoint, fmt.Errorf("err (%s): header Content-Type is not JSON (%s)", contentType, resp.Status)
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, endpoint, fmt.Errorf("cannot read body")
	}

	if resp.StatusCode >= 400 {
		return nil, endpoint, unmarshalOpenstackError(rbody, resp.Status)
	}

	var authRet AuthToken
	err = json.Unmarshal(rbody, &authRet)
	if err != nil {
		return nil, endpoint, err
	}

	for _, sc := range authRet.Access.ServiceCatalog {
		log.Notice("OpenStack> Looking for service 'swift' (got %s)\n", sc.Name)
		if sc.Name == "swift" {
			log.Notice("OpenStack> Found swift !\n")
			for _, e := range sc.Endpoints {
				log.Notice("OpenStack> Looking for region P19 service 'swift' (got %s)\n", e.Region)
				if e.Region == "P19" {
					log.Warning("OpenStack> Got Swift in P19 !\n")
					endpoint = sc.Endpoints[0].PublicURL
				}
			}
		}
	}
	if endpoint == "" {
		return nil, "", fmt.Errorf("swift endpoint not found")
	}

	return &authRet.Access.Token, endpoint, nil
}

/*{"error": {"message": "The request you have made requires authentication.", "code": 401, "title": "Unauthorized"}}*/
type openstackError struct {
	Error struct {
		Message string `json:"error"`
		Code    int    `json:"code"`
		Title   string `json:"title"`
	} `json:"error"`
}

func unmarshalOpenstackError(data []byte, status string) error {
	operror := openstackError{}
	err := json.Unmarshal(data, &operror)
	if err != nil {
		return fmt.Errorf("%s", status)
	}

	if operror.Error.Code == 0 {
		return fmt.Errorf("%s", status)
	}

	return fmt.Errorf("%d: %s", operror.Error.Code, operror.Error.Message)
}
