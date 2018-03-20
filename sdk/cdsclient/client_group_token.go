package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

// GroupGenerateToken generate a token in a group
func (c *client) GroupGenerateToken(groupName, expiration, description string) (*sdk.Token, error) {
	path := fmt.Sprintf("/group/%s/token", url.QueryEscape(groupName))
	desc := struct {
		Description string `json:"description"`
		Expiration  string `json:"expiration"`
	}{Description: description, Expiration: expiration}

	var token sdk.Token
	if _, err := c.PostJSON(path, desc, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// GroupDeleteToken delete a token in a group given its id
func (c *client) GroupDeleteToken(groupName string, tokenID int64) error {
	path := fmt.Sprintf("/group/%s/token/%d", url.QueryEscape(groupName), tokenID)

	if _, err := c.DeleteJSON(path, nil); err != nil {
		return err
	}
	return nil
}

// GroupListToken list tokens in a group
func (c *client) GroupListToken(groupName string) ([]sdk.Token, error) {
	path := fmt.Sprintf("/group/%s/token", url.QueryEscape(groupName))

	tokens := []sdk.Token{}
	_, err := c.GetJSON(path, &tokens)

	return tokens, err
}
