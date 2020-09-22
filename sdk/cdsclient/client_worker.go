package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkerList(ctx context.Context) ([]sdk.Worker, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := []sdk.Worker{}
	if _, err := c.GetJSON(ctx, "/worker", &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) WorkerGet(ctx context.Context, name string, mods ...RequestModifier) (*sdk.Worker, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var wrk sdk.Worker
	if _, err := c.GetJSON(ctx, fmt.Sprintf("/worker/%s", name), &wrk, mods...); err != nil {
		return nil, err
	}
	return &wrk, nil
}

func (c *client) WorkerUnregister(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := c.PostJSON(ctx, "/auth/consumer/worker/signout", nil, nil); err != nil {
		return err
	}
	return nil
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

func (c *client) WorkerRegister(ctx context.Context, authToken string, form sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var w sdk.Worker

	var jwtHeader = func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	_, headers, code, err := c.RequestJSON(ctx, "POST", "/auth/consumer/worker/signin", form, &w, jwtHeader)
	if code == http.StatusUnauthorized {
		return nil, false, sdk.ErrUnauthorized
	}
	if err != nil {
		return nil, false, err
	}
	c.config.SessionToken = headers.Get("X-CDS-JWT")

	if c.config.Verbose {
		fmt.Printf("Registering session %s for worker %s\n", sdk.StringFirstN(c.config.SessionToken, 12), w.Name)
	}

	return &w, w.Uptodate, nil
}

func (c *client) WorkerSetStatus(ctx context.Context, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var uri string
	switch status {
	case sdk.StatusChecking:
		uri = fmt.Sprintf("/worker/checking")
	case sdk.StatusWaiting:
		uri = fmt.Sprintf("/worker/waiting")
	default:
		return fmt.Errorf("Unsupported status: %s", status)
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
