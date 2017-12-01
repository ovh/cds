package cdsclient

import (
	"fmt"
)

func (c *client) ConfigUser() (map[string]string, error) {
	var res map[string]string
	code, err := c.GetJSON("/config/user", &res)
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
