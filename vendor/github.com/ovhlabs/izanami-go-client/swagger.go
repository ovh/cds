package client

// Get swagger.json datas
func (c *SwaggerClient) Get() (string, error) {
	body, errGet := c.client.get("/swagger.json", nil)
	if errGet != nil {
		return "", errGet
	}
	return string(body), nil
}
