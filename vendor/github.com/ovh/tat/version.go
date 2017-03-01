package tat

// Version returns Tat Engine version
func (c *Client) Version() ([]byte, error) {
	return c.reqWant("GET", 200, "/version", nil)
}
