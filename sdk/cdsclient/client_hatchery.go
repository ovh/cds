package cdsclient

import (
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) HatcheryRegister(h sdk.Hatchery) (*sdk.Hatchery, error) {
	var hreceived sdk.Hatchery
	h.UID = c.config.Token
	code, err := c.PostJSON("/hatchery", &h, &hreceived)
	if code == http.StatusUnauthorized {
		return nil, sdk.ErrUnauthorized
	}
	if code > 300 && err == nil {
		return nil, fmt.Errorf("HTTP %d", code)
	} else if err != nil {
		return nil, err
	}

	c.isHatchery = true
	c.config.Hash = hreceived.UID

	return &hreceived, nil
}
