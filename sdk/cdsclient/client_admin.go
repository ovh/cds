package cdsclient

import (
	"bytes"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) Services() ([]sdk.Service, error) {
	srvs := []sdk.Service{}
	if _, err := c.GetJSON("/admin/services", &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) ServicesByName(name string) (*sdk.Service, error) {
	srv := sdk.Service{}
	if _, err := c.GetJSON("/admin/service/"+name, &srv); err != nil {
		return nil, err
	}
	return &srv, nil
}

func (c *client) ServicesByType(stype string) ([]sdk.Service, error) {
	srvs := []sdk.Service{}
	if _, err := c.GetJSON("/admin/services?type="+stype, &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) ServiceCallGET(stype string, sname string, query string) ([]byte, error) {
	btes, _, _, err := c.Request("GET", "/admin/services/call?type="+stype+"&name="+sname+"&query="+url.QueryEscape(query), nil)
	return btes, err
}

func (c *client) ServiceCallPOST(stype string, sname string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request("POST", "/admin/services/call?type="+stype+"&name="+sname+"&query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallPUT(stype string, sname string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request("PUT", "/admin/services/call?type="+stype+"&name="+sname+"&query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallDELETE(stype string, sname string, query string) error {
	_, _, _, err := c.Request("DELETE", "/admin/services/call?type="+stype+"&name="+sname+"&query="+url.QueryEscape(query), nil)
	return err
}
