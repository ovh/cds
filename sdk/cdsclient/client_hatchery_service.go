package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *hatcheryClient) Heartbeat(ctx context.Context, mon *sdk.MonitoringStatus) error {
	if _, err := c.PostJSON(ctx, "/v2/hatchery/heartbeat", &mon, nil); err != nil {
		return err
	}
	return nil
}

func (c *hatcheryClient) GetWorkerModel(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, workerModelName string, mods ...RequestModifier) (*sdk.V2WorkerModel, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workermodel/%s", projKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), workerModelName)
	var wm sdk.V2WorkerModel
	if _, err := c.GetJSON(ctx, path, &wm, mods...); err != nil {
		return nil, err
	}
	return &wm, nil
}
