package cdsclient

import (
	"bytes"
	"context"
	"fmt"
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
