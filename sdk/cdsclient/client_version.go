package cdsclient

import (
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) Version() (*sdk.Version, error) {
	v := &sdk.Version{}

	code, err := c.GetJSON("/mon/version", v)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("Error %d", code)
	}
	return v, err
}
