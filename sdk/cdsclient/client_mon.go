package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) MonStatus() ([]string, error) {
	res := []string{}
	code, err := c.GetJSON("/mon/status", &res)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) MonDBTimes() (*sdk.MonDBTimes, error) {
	monDBTimes := sdk.MonDBTimes{}
	code, err := c.GetJSON("/mon/db/times", &monDBTimes)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return &monDBTimes, nil
}
