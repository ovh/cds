package cdsclient

import "context"

func (c *client) ConfigUser() (map[string]string, error) {
	var res map[string]string
	if _, err := c.GetJSON(context.Background(), "/config/user", &res); err != nil {
		return nil, err
	}

	return res, nil
}
