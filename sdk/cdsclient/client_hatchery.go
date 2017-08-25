package cdsclient

import (
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) HatcheryRegister(h sdk.Hatchery) (*sdk.Hatchery, bool, error) {
	var hreceived sdk.Hatchery
	h.UID = c.config.Token
	code, err := c.PostJSON("/hatchery", &h, &hreceived)
	if code == http.StatusUnauthorized {
		return nil, false, sdk.ErrUnauthorized
	}
	if code > 300 && err == nil {
		return nil, false, fmt.Errorf("HTTP %d", code)
	} else if err != nil {
		return nil, false, err
	}

	c.isHatchery = true
	c.config.Hash = hreceived.UID

	return &hreceived, hreceived.Uptodate, nil
}
