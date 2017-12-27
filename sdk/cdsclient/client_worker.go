package cdsclient

import (
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkerList() ([]sdk.Worker, error) {
	p := []sdk.Worker{}
	if _, err := c.GetJSON("/worker", &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) WorkerRegister(r sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error) {
	var w sdk.Worker
	code, err := c.PostJSON("/worker", r, &w)
	if code == http.StatusUnauthorized {
		return nil, false, sdk.ErrUnauthorized
	}
	if code > 300 && err == nil {
		return nil, false, fmt.Errorf("HTTP %d", code)
	} else if err != nil {
		return nil, false, err
	}

	c.isWorker = true
	c.config.Hash = w.ID

	return &w, w.Uptodate, nil
}

func (c *client) WorkerSetStatus(s sdk.Status) error {
	var uri string
	switch s {
	case sdk.StatusChecking:
		uri = fmt.Sprintf("/worker/checking")
	case sdk.StatusWaiting:
		uri = fmt.Sprintf("/worker/waiting")
	default:
		return fmt.Errorf("Unsupported status : %s", s.String())
	}

	code, err := c.PostJSON(uri, nil, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}
