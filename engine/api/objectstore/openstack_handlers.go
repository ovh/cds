package objectstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/log"
)

func (ops *OpenstackStore) refreshTokenRoutine(c context.Context) {
	tick := time.NewTicker(20 * time.Hour).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting refreshTokenRoutine: %v", c.Err())
				return
			}
		case <-tick:
			tk, endpoint, err := getToken(ops.user, ops.password, ops.address, ops.tenant, ops.region)
			if err != nil {
				log.Error("refreshTokenRoutine> Cannot refresh token: %s\n", err)
				continue
			}
			ops.token = tk
			ops.endpoint = endpoint
		}
	}
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

func getToken(user, password, url, project, region string) (*Token, string, error) {
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
		log.Debug("OpenStack> Looking for service 'swift' (got %s)\n", sc.Name)
		if sc.Name == "swift" {
			log.Debug("OpenStack> Found swift !\n")
			for _, e := range sc.Endpoints {
				log.Debug("OpenStack> Looking for region %s service 'swift' (got %s)\n", region, e.Region)
				if e.Region == region {
					log.Debug("OpenStack> Got Swift in %s !\n", region)
					endpoint = e.PublicURL

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
