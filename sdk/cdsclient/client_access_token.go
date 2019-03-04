package cdsclient

import (
	"context"
	"strconv"

	"github.com/ovh/cds/sdk"
)

func (c *client) AccessTokenListByUser(username string) ([]sdk.AccessToken, error) {
	u, err := c.UserGet(username)
	if err != nil {
		return nil, err
	}

	var tokens []sdk.AccessToken
	if _, err := c.GetJSON(context.Background(), "/accesstoken/user/"+strconv.FormatInt(u.ID, 10), &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (c *client) AccessTokenListByGroup(groups ...string) ([]sdk.AccessToken, error) {
	var allTokens []sdk.AccessToken

	for _, s := range groups {
		g, err := c.GroupGet(s)
		if err != nil {
			return nil, err
		}

		var tokens []sdk.AccessToken
		if _, err := c.GetJSON(context.Background(), "/accesstoken/group/"+strconv.FormatInt(g.ID, 10), &tokens); err != nil {
			return nil, err
		}

		allTokens = append(allTokens, tokens...)
	}

	return allTokens, nil
}
func (c *client) AccessTokenDelete(id string) error {
	_, err := c.DeleteJSON(context.Background(), "/accesstoken/"+id, nil)
	return err
}

func (c *client) AccessTokenCreate(request sdk.AccessTokenRequest) (sdk.AccessToken, string, error) {
	var t sdk.AccessToken
	_, headers, _, err := c.RequestJSON(context.Background(), "POST", "/accesstoken", request, &t)
	if err != nil {
		return t, "", err
	}
	jwt := headers.Get("X-CDS-JWT")
	return t, jwt, nil
}
func (c *client) AccessTokenRegen(id string) (sdk.AccessToken, string, error) {
	var t sdk.AccessToken
	_, headers, _, err := c.RequestJSON(context.Background(), "PUT", "/accesstoken/"+id, nil, &t)
	if err != nil {
		return t, "", err
	}
	jwt := headers.Get("X-CDS-JWT")
	return t, jwt, nil
}
