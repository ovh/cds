package cdsclient

import (
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

func (c *client) MonDBTimes() (*sdk.MonDBTimes, error) {
	monDBTimes := sdk.MonDBTimes{}
	if _, err := c.GetJSON("/mon/db/times", &monDBTimes); err != nil {
		return nil, err
	}
	return &monDBTimes, nil
}
