package cdsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) AdminDatabaseMigrationDelete(id string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/admin/database/migration/delete/"+url.QueryEscape(id), nil)
	return err
}

func (c *client) AdminDatabaseMigrationsList() ([]sdk.DatabaseMigrationStatus, error) {
	dlist := []sdk.DatabaseMigrationStatus{}
	if _, err := c.GetJSON(context.Background(), "/admin/database/migration", &dlist); err != nil {
		return nil, err
	}
	return dlist, nil
}

func (c *client) AdminDatabaseMigrationUnlock(id string) error {
	_, _, _, err := c.Request(context.Background(), "POST", "/admin/database/migration/unlock/"+url.QueryEscape(id), nil)
	return err
}

func (c *client) AdminCDSMigrationCancel(id int64) error {
	_, _, _, err := c.Request(context.Background(), "POST", fmt.Sprintf("/admin/cds/migration/%d/cancel", id), nil)
	return err
}

func (c *client) AdminCDSMigrationReset(id int64) error {
	_, _, _, err := c.Request(context.Background(), "POST", fmt.Sprintf("/admin/cds/migration/%d/todo", id), nil)
	return err
}

func (c *client) AdminCDSMigrationList() ([]sdk.Migration, error) {
	var migrations []sdk.Migration
	if _, err := c.GetJSON(context.Background(), "/admin/cds/migration", &migrations); err != nil {
		return nil, err
	}
	return migrations, nil
}

func (c *client) Services() ([]sdk.Service, error) {
	srvs := []sdk.Service{}
	if _, err := c.GetJSON(context.Background(), "/admin/services", &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) ServicesByName(name string) (*sdk.Service, error) {
	srv := sdk.Service{}
	if _, err := c.GetJSON(context.Background(), "/admin/service/"+name, &srv); err != nil {
		return nil, err
	}
	return &srv, nil
}

func (c *client) ServicesByType(stype string) ([]sdk.Service, error) {
	srvs := []sdk.Service{}
	if _, err := c.GetJSON(context.Background(), "/admin/services?type="+stype, &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) ServiceNameCallGET(name string, query string) ([]byte, error) {
	btes, _, _, err := c.Request(context.Background(), "GET", "/admin/services/call?name="+name+"&query="+url.QueryEscape(query), nil)
	return btes, err
}

func (c *client) ServiceDelete(name string) error {
	_, err := c.DeleteJSON(context.Background(), "/admin/service/"+name, nil)
	return err
}

func (c *client) ServiceCallGET(stype string, query string) ([]byte, error) {
	btes, _, _, err := c.Request(context.Background(), "GET", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(query), nil)
	return btes, err
}

func (c *client) ServiceCallPOST(stype string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request(context.Background(), "POST", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallPUT(stype string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request(context.Background(), "PUT", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallDELETE(stype string, query string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(query), nil)
	return err
}

func (c *client) ServiceGetJSON(ctx context.Context, stype, path string, out interface{}) (int, error) {
	btes, _, code, err := c.Request(ctx, "GET", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(path), nil)
	if err != nil {
		return code, err
	}
	if err := sdk.JSONUnmarshal(btes, out); err != nil {
		return code, newError(err)
	}
	return code, nil
}

func (c *client) ServicePostJSON(ctx context.Context, stype, path string, in, out interface{}) (int, error) {
	var inBtes []byte
	if in != nil {
		var err error
		inBtes, err = json.Marshal(in)
		if err != nil {
			return 0, newError(err)
		}
	}

	btes, _, code, err := c.Request(ctx, "POST", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(path), bytes.NewReader(inBtes))
	if err != nil {
		return code, err
	}

	if len(btes) > 0 {
		if err := sdk.JSONUnmarshal(btes, out); err != nil {
			return code, newError(err)
		}
	}

	return code, nil
}

func (c *client) ServicePutJSON(ctx context.Context, stype, path string, in, out interface{}) (int, error) {
	var inBtes []byte
	if in != nil {
		var err error
		inBtes, err = json.Marshal(in)
		if err != nil {
			return 0, newError(err)
		}
	}

	btes, _, code, err := c.Request(ctx, "PUT", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(path), bytes.NewReader(inBtes))
	if err != nil {
		return code, err
	}

	if len(btes) > 0 {
		if err := sdk.JSONUnmarshal(btes, out); err != nil {
			return code, newError(err)
		}
	}
	return code, nil
}

func (c *client) ServiceDeleteJSON(ctx context.Context, stype, path string, out interface{}) (int, error) {
	btes, _, code, err := c.Request(ctx, "DELETE", "/admin/services/call?type="+stype+"&query="+url.QueryEscape(path), nil)
	if err != nil {
		return code, err
	}

	if err := sdk.JSONUnmarshal(btes, out); err != nil {
		return code, newError(err)
	}
	return code, nil
}

func (c *client) switchServiceCallFunc(service string, method, path string, in, out interface{}) func() (int, error) {
	switch method {
	case http.MethodGet:
		switch service {
		case sdk.TypeAPI:
			return func() (int, error) {
				return c.GetJSON(context.Background(), path, out)
			}
		default:
			return func() (int, error) {
				return c.ServiceGetJSON(context.Background(), service, path, out)
			}
		}
	case http.MethodPost:
		switch service {
		case sdk.TypeAPI:
			return func() (int, error) {
				return c.PostJSON(context.Background(), path, in, out)
			}
		default:
			return func() (int, error) {
				return c.ServicePostJSON(context.Background(), service, path, in, out)
			}
		}
	case http.MethodPut:
		switch service {
		case sdk.TypeAPI:
			return func() (int, error) {
				return c.PutJSON(context.Background(), path, in, out)
			}
		default:
			return func() (int, error) {
				return c.ServicePutJSON(context.Background(), service, path, in, out)
			}
		}
	case http.MethodDelete:
		switch service {
		case sdk.TypeAPI:
			return func() (int, error) {
				return c.DeleteJSON(context.Background(), path, out)
			}
		default:
			return func() (int, error) {
				return c.ServiceDeleteJSON(context.Background(), service, path, out)
			}
		}
	}
	return nil
}

func (c *client) AdminDatabaseSignaturesResume(service string) (sdk.CanonicalFormUsageResume, error) {
	var res = sdk.CanonicalFormUsageResume{}
	var f = c.switchServiceCallFunc(service, http.MethodGet, "/admin/database/signature", nil, &res)
	_, err := f()
	return res, err
}

func (c *client) AdminDatabaseSignaturesRollEntity(service string, e string) error {
	resume, err := c.AdminDatabaseSignaturesResume(service)
	if err != nil {
		return err
	}

	if _, has := resume[e]; !has {
		return errors.New("unkown entity")
	}

	for _, s := range resume[e] {
		url := fmt.Sprintf("/admin/database/signature/%s/%s", e, s.Signer)
		var pks []string
		var f = c.switchServiceCallFunc(service, http.MethodGet, url, nil, &pks)
		if _, err := f(); err != nil {
			return err
		}

		for _, pk := range pks {
			url := fmt.Sprintf("/admin/database/signature/%s/roll/%s", e, pk)
			var f = c.switchServiceCallFunc(service, http.MethodPost, url, nil, nil)
			if _, err := f(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *client) AdminDatabaseSignaturesRollAllEntities(service string) error {
	resume, err := c.AdminDatabaseSignaturesResume(service)
	if err != nil {
		return err
	}

	for e := range resume {
		if err := c.AdminDatabaseSignaturesRollEntity(service, e); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) AdminDatabaseListEncryptedEntities(service string) ([]string, error) {
	var res []string
	var f = c.switchServiceCallFunc(service, http.MethodGet, "/admin/database/encryption", nil, &res)
	_, err := f()
	return res, err
}

func (c *client) AdminDatabaseRollEncryptedEntity(service string, e string) error {
	url := fmt.Sprintf("/admin/database/encryption/%s", e)
	var pks []string

	var f = c.switchServiceCallFunc(service, http.MethodGet, url, nil, &pks)
	if _, err := f(); err != nil {
		return err
	}

	for _, pk := range pks {
		url := fmt.Sprintf("/admin/database/encryption/%s/roll/%s", e, pk)
		var f = c.switchServiceCallFunc(service, http.MethodPost, url, nil, nil)
		if _, err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) AdminDatabaseRollAllEncryptedEntities(service string) error {
	entities, err := c.AdminDatabaseListEncryptedEntities(service)
	if err != nil {
		return err
	}
	for _, e := range entities {
		if err := c.AdminDatabaseRollEncryptedEntity(service, e); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) AdminWorkflowUpdateMaxRuns(projectKey string, workflowName string, maxRuns int64) error {
	request := sdk.UpdateMaxRunRequest{MaxRuns: maxRuns}
	url := fmt.Sprintf("/project/%s/workflows/%s/retention/maxruns", projectKey, workflowName)
	if _, err := c.PostJSON(context.Background(), url, &request, nil); err != nil {
		return err
	}
	return nil
}
