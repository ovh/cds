package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) V2WorkerGet(ctx context.Context, name string, mods ...RequestModifier) (*sdk.V2Worker, error) {
	var worker sdk.V2Worker
	url := fmt.Sprintf("/v2/worker/" + name)
	if _, err := c.GetJSON(ctx, url, &worker, mods...); err != nil {
		return nil, err
	}
	return &worker, nil
}

func (c *client) V2QueueWorkerTakeJob(ctx context.Context, region, runJobID string) (*sdk.V2TakeJobResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var takenJob sdk.V2TakeJobResponse
	url := fmt.Sprintf("/v2/queue/%s/job/%s/worker/take", region, runJobID)
	if _, err := c.PostJSON(ctx, url, nil, &takenJob); err != nil {
		return nil, err
	}
	return &takenJob, nil
}

func (c *client) V2WorkerRefresh(ctx context.Context, region, runJobID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := fmt.Sprintf("/v2/queue/%s/job/%s/worker/refresh", region, runJobID)
	if _, err := c.PostJSON(ctx, url, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) V2WorkerRegister(ctx context.Context, authToken string, form sdk.WorkerRegistrationForm, region, runJobID string) (*sdk.V2Worker, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var w sdk.V2Worker
	var jwtHeader = func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	path := fmt.Sprintf("/v2/queue/%s/job/%s/worker/signin", region, runJobID)
	_, headers, code, err := c.RequestJSON(ctx, "POST", path, form, &w, jwtHeader)
	if code == http.StatusUnauthorized {
		return nil, sdk.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	c.config.SessionToken = headers.Get("X-CDS-JWT")

	if c.config.Verbose {
		fmt.Printf("Registering session %s for worker %s\n", sdk.StringFirstN(c.config.SessionToken, 12), w.Name)
	}
	return &w, nil
}

func (c *client) V2WorkerUnregister(ctx context.Context, region, runJobID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	path := fmt.Sprintf("/v2/queue/%s/job/%s/worker/signout", region, runJobID)
	if _, err := c.PostJSON(ctx, path, nil, nil); err != nil {
		return err
	}
	return nil
}
