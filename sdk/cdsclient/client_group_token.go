package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupGenerateToken(groupName, expiration, description string) (*sdk.Token, error) {
	path := fmt.Sprintf("/group/%s/token/%s", url.QueryEscape(groupName), expiration)
	desc := struct {
		Description string `json:"description"`
	}{Description: description}

	var token sdk.Token
	if _, err := c.PostJSON(path, desc, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
