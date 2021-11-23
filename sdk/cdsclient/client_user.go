package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) UserList(ctx context.Context) ([]sdk.AuthentifiedUser, error) {
	res := []sdk.AuthentifiedUser{}
	if _, err := c.GetJSON(ctx, "/user", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) UserGet(ctx context.Context, username string) (*sdk.AuthentifiedUser, error) {
	var res sdk.AuthentifiedUser
	if _, err := c.GetJSON(ctx, "/user/"+url.QueryEscape(username), &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) UserGetMe(ctx context.Context) (*sdk.AuthentifiedUser, error) {
	var res sdk.AuthentifiedUser
	if _, err := c.GetJSON(ctx, "/user/me", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) UserGetGroups(ctx context.Context, username string) (map[string][]sdk.Group, error) {
	res := map[string][]sdk.Group{}
	if _, err := c.GetJSON(ctx, "/user/"+url.QueryEscape(username)+"/groups", &res); err != nil {
		return nil, err
	}
	return res, nil
}

// UpdateFavorite Update favorites (add or delete) return updated workflow or project
func (c *client) UpdateFavorite(ctx context.Context, params sdk.FavoriteParams) (interface{}, error) {
	switch params.Type {
	case "workflow":
		var wf sdk.Workflow
		if _, err := c.PostJSON(ctx, "/user/favorite", params, &wf); err != nil {
			return wf, err
		}
		return wf, nil
	case "project":
		var proj sdk.Project
		if _, err := c.PostJSON(ctx, "/user/favorite", params, &proj); err != nil {
			return proj, err
		}
		return proj, nil
	}

	var res interface{}
	if _, err := c.PostJSON(ctx, "/user/favorite", params, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) UserGetSchema(ctx context.Context) (sdk.SchemaResponse, error) {
	var res sdk.SchemaResponse
	if _, err := c.GetJSON(ctx, "/user/schema", &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) UserUpdate(ctx context.Context, username string, u *sdk.AuthentifiedUser) error {
	if _, err := c.PutJSON(ctx, "/user/"+url.QueryEscape(username), u, u); err != nil {
		return err
	}
	return nil
}
