package cdsclient

func (c *client) ConfigUser() (map[string]string, error) {
	var res map[string]string
	if _, err := c.GetJSON("/config/user", &res); err != nil {
		return nil, err
	}

	return res, nil
}
