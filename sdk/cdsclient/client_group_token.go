package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupGenerateToken(groupName, expiration string) (*sdk.Token, error) {
	path := fmt.Sprintf("/group/%s/token/%s", url.QueryEscape(groupName), expiration)
	var token sdk.Token
	if _, err := c.PostJSON(path, nil, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
