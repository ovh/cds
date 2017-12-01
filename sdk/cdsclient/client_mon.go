package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) MonStatus() ([]string, error) {
	res := []string{}
	if _, err := c.GetJSON("/mon/status", &res); err != nil {
		return nil, err
	}

	return res, nil
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
