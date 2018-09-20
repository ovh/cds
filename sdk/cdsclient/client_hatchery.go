package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

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
