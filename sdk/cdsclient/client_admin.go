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

func (c *client) ServicesByType(s string) ([]sdk.Service, error) {
	srvs := []sdk.Service{}
	if _, err := c.GetJSON("/admin/services/"+s, &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) ServiceCallGET(s string, query string) ([]byte, error) {
	btes, _, _, err := c.Request("GET", "/admin/services/"+s+"/call?query="+url.QueryEscape(query), nil)
	return btes, err
}

func (c *client) ServiceCallPOST(s string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request("POST", "/admin/services/"+s+"/call?query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallPUT(s string, query string, body []byte) ([]byte, error) {
	rBody := bytes.NewReader(body)
	btes, _, _, err := c.Request("PUT", "/admin/services/"+s+"/call?query="+url.QueryEscape(query), rBody)
	return btes, err
}

func (c *client) ServiceCallDELETE(s string, query string) error {
	_, _, _, err := c.Request("DELETE", "/admin/services/"+s+"/call?query="+url.QueryEscape(query), nil)
	return err
}
