package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupGenerateToken(groupName, expiration string) (*sdk.Token, error) {
	path := fmt.Sprintf("/group/%s/token/%s", url.QueryEscape(groupName), expiration)
	var token sdk.Token
	code, err := c.PostJSON(path, nil, &token)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}
