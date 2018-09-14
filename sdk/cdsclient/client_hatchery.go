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
		return nil, fmt.Errorf("HatcheryRegister> HTTP %d", code)
	} else if err != nil {
		return nil, sdk.WrapError(err, "HatcheryRegister> Error")
	}

	c.isHatchery = true
	c.config.Hash = hreceived.UID

	return &hreceived, nil
}

func (c *client) HatcheryCount(workflowNodeRunID int64) (int64, error) {
	var hatcheriesCount int64
	code, err := c.GetJSON(fmt.Sprintf("/hatchery/count/%d", workflowNodeRunID), &hatcheriesCount)
	if code > 300 && err == nil {
		return hatcheriesCount, fmt.Errorf("HatcheryCount> HTTP %d", code)
	} else if err != nil {
		return hatcheriesCount, sdk.WrapError(err, "HatcheryCount> Error")
	}
	return hatcheriesCount, nil
}
