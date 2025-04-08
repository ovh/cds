package cdsclient

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) AdminDatabaseMigrationList(service string) ([]sdk.DatabaseMigrationStatus, error) {
	dlist := []sdk.DatabaseMigrationStatus{}
	var f = c.switchServiceCallFunc(service, http.MethodGet, "/admin/database/migration", nil, &dlist)
	_, err := f()
	return dlist, err
}

func (c *client) AdminDatabaseMigrationDelete(service string, id string) error {
	path := fmt.Sprintf("/admin/database/migration/delete/%s", url.QueryEscape(id))
	var f = c.switchServiceCallFunc(service, http.MethodDelete, path, nil, nil)
	_, err := f()
	return err
}

func (c *client) AdminDatabaseMigrationUnlock(service string, id string) error {
	path := fmt.Sprintf("/admin/database/migration/unlock/%s", url.QueryEscape(id))
	var f = c.switchServiceCallFunc(service, http.MethodPost, path, nil, nil)
	_, err := f()
	return err
}

func (c *client) AdminDatabaseEntityList(service string) ([]sdk.DatabaseEntity, error) {
	var res []sdk.DatabaseEntity
	var f = c.switchServiceCallFunc(service, http.MethodGet, "/admin/database/entity", nil, &res)
	_, err := f()
	return res, err
}

func (c *client) AdminDatabaseEntity(service string, e string, mods ...RequestModifier) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/admin/database/entity/%s", e), nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	for _, m := range mods {
		m(req)
	}
	var pks []string
	var f = c.switchServiceCallFunc(service, http.MethodGet, req.URL.String(), nil, &pks)
	_, err = f()
	return pks, err
}

func (c *client) AdminDatabaseEntityInfo(service string, e string, pks []string) ([]sdk.DatabaseEntityInfo, error) {
	var res []sdk.DatabaseEntityInfo
	url := fmt.Sprintf("/admin/database/entity/%s/info", e)
	var f = c.switchServiceCallFunc(service, http.MethodPost, url, pks, &res)
	_, err := f()
	return res, err
}

func (c *client) AdminDatabaseEntityRoll(service string, e string, pks []string, mods ...RequestModifier) ([]sdk.DatabaseEntityInfo, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/admin/database/entity/%s/roll", e), nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	for _, m := range mods {
		m(req)
	}
	var res []sdk.DatabaseEntityInfo
	var f = c.switchServiceCallFunc(service, http.MethodPost, req.URL.String(), pks, &res)
	_, err = f()
	return res, err
}
