package cdsclient

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) MonStatus() (*sdk.MonitoringStatus, error) {
	monStatus := sdk.MonitoringStatus{}
	if _, err := c.GetJSON("/mon/status", &monStatus); err != nil {
		return nil, err
	}
	return &monStatus, nil
}

func (c *client) MonVersion() (*sdk.Version, error) {
	monVersion := sdk.Version{}
	if _, err := c.GetJSON("/mon/version", &monVersion); err != nil {
		return nil, err
	}
	return &monVersion, nil
}

func (c *client) MonDBMigrate() ([]sdk.MonDBMigrate, error) {
	monDBMigrate := []sdk.MonDBMigrate{}
	if _, err := c.GetJSON("/mon/db/migrate", &monDBMigrate); err != nil {
		return nil, err
	}
	return monDBMigrate, nil
}

func (c *client) MonErrorsGet(uuid string) (*sdk.Error, error) {
	res, _, _, err := c.Request("GET", fmt.Sprintf("/mon/errors/%s", uuid), nil)
	if err != nil {
		return nil, err
	}

	var sdkError sdk.Error
	if err := json.Unmarshal(res, &sdkError); err != nil {
		return nil, err
	}

	return &sdkError, nil
}
