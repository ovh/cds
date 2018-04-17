package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ServiceRegister(s sdk.Service) (string, error) {
	code, err := c.PostJSON("/services/register", &s, &s)
	if code != 201 && code != 200 {
		if err == nil {
			return "", fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return "", err
	}
	c.isService = true

	// TODO remove that when hatchery will be real CDS Services
	if s.Type != "hatchery" {
		c.config.Hash = s.Hash
	}

	return s.Hash, nil
}
