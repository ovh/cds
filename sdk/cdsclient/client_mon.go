package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) MonStatus() (*sdk.MonitoringStatus, error) {
	monStatus := sdk.MonitoringStatus{}
	if _, err := c.GetJSON(context.Background(), "/mon/status", &monStatus); err != nil {
		return nil, err
	}
	return &monStatus, nil
}

func (c *client) MonVersion() (*sdk.Version, error) {
	monVersion := sdk.Version{}
	if _, err := c.GetJSON(context.Background(), "/mon/version", &monVersion); err != nil {
		return nil, err
	}
	return &monVersion, nil
}

func (c *client) MonDBMigrate() ([]sdk.MonDBMigrate, error) {
	monDBMigrate := []sdk.MonDBMigrate{}
	if _, err := c.GetJSON(context.Background(), "/mon/db/migrate", &monDBMigrate); err != nil {
		return nil, err
	}
	return monDBMigrate, nil
}

func (c *client) MonErrorsGet(requestID string) ([]sdk.Error, error) {
	res, _, _, err := c.Request(context.Background(), "GET", fmt.Sprintf("/mon/errors/%s", requestID), nil)
	if err != nil {
		return nil, err
	}

	var errs []sdk.Error
	if err := sdk.JSONUnmarshal(res, &errs); err != nil {
		return nil, err
	}

	return errs, nil
}
