package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkerList(ctx context.Context) ([]sdk.Worker, error) {
	p := []sdk.Worker{}
	if _, err := c.GetJSON(ctx, "/worker", &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) WorkerDisable(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := fmt.Sprintf("/worker/%s/disable", id)
	if _, err := c.PostJSON(ctx, url, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkerRefresh(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := fmt.Sprintf("/worker/refresh")
	if _, err := c.PostJSON(ctx, url, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkerRegister(ctx context.Context, form sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var w sdk.Worker
	code, err := c.PostJSON(ctx, "/worker", form, &w)
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

func (c *client) WorkerSetStatus(ctx context.Context, status sdk.Status) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var uri string
	switch status {
	case sdk.StatusChecking:
		uri = fmt.Sprintf("/worker/checking")
	case sdk.StatusWaiting:
		uri = fmt.Sprintf("/worker/waiting")
	default:
		return fmt.Errorf("Unsupported status: %s", status.String())
	}

	code, err := c.PostJSON(ctx, uri, nil, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}
