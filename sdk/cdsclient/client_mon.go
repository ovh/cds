package cdsclient

import (
	"fmt"
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
