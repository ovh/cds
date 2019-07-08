package izanami

// Get swagger.json datas
func (c *SwaggerClient) Get() (string, error) {
	body, err := c.client.get("/swagger.json", nil)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
