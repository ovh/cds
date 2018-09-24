package cdsclient

import (
	"fmt"
	"runtime"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (c *client) GetService() *sdk.Service {
	return c.service
}

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
	c.config.Hash = s.Hash
	c.service = &s

	if !s.Uptodate {
		log.Warning("-=-=-=-=- Please update your cds engine binary - current version:%s %s %s -=-=-=-=-",
			sdk.VERSION, runtime.GOOS, runtime.GOARCH)
	}

	return s.Hash, nil
}
