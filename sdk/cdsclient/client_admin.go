package cdsclient

import (
	"bytes"
	"context"
	"errors"
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

func (c *client) AdminDatabaseSignaturesResume() (sdk.CanonicalFormUsageResume, error) {
	var res = sdk.CanonicalFormUsageResume{}
	_, err := c.GetJSON(context.Background(), "/admin/database/signature", &res)
	return res, err
}

func (c *client) AdminDatabaseSignaturesRollEntity(e string) error {
	resume, err := c.AdminDatabaseSignaturesResume()
	if err != nil {
		return err
	}

	if _, has := resume[e]; !has {
		return errors.New("unkown entity")
	}

	for _, s := range resume[e] {
		url := fmt.Sprintf("/admin/database/signature/%s/%s", e, s.Signer)
		var pks []string
		if _, err := c.GetJSON(context.Background(), url, &pks); err != nil {
			return err
		}

		for _, pk := range pks {
			url := fmt.Sprintf("/admin/database/signature/%s/roll/%s", e, pk)
			if _, err := c.PostJSON(context.Background(), url, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *client) AdminDatabaseSignaturesRollAllEntities() error {
	resume, err := c.AdminDatabaseSignaturesResume()
	if err != nil {
		return err
	}

	for e := range resume {
		if err := c.AdminDatabaseSignaturesRollEntity(e); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) AdminDatabaseListEncryptedEntities() ([]string, error) {
	var res []string
	_, err := c.GetJSON(context.Background(), "/admin/database/encryption", &res)
	return res, err
}

func (c *client) AdminDatabaseRollEncryptedEntity(e string) error {
	url := fmt.Sprintf("/admin/database/encryption/%s", e)
	var pks []string
	if _, err := c.GetJSON(context.Background(), url, &pks); err != nil {
		return err
	}

	for _, pk := range pks {
		url := fmt.Sprintf("/admin/database/encryption/%s/roll/%s", e, pk)
		if _, err := c.PostJSON(context.Background(), url, nil, nil); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) AdminDatabaseRollAllEncryptedEntities() error {
	entities, err := c.AdminDatabaseListEncryptedEntities()
	if err != nil {
		return err
	}
	for _, e := range entities {
		if err := c.AdminDatabaseRollEncryptedEntity(e); err != nil {
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
