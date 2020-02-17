package cdsclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) UserList() ([]sdk.AuthentifiedUser, error) {
	res := []sdk.AuthentifiedUser{}
	if _, err := c.GetJSON(context.Background(), "/user", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) UserGet(username string) (*sdk.AuthentifiedUser, error) {
	var res sdk.AuthentifiedUser
	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username), &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) UserGetMe() (*sdk.AuthentifiedUser, error) {
	var res sdk.AuthentifiedUser
	if _, err := c.GetJSON(context.Background(), "/user/me", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) UserGetGroups(username string) (map[string][]sdk.Group, error) {
	res := map[string][]sdk.Group{}
	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username)+"/groups", &res); err != nil {
		return nil, err
	}
	return res, nil
}

// UpdateFavorite Update favorites (add or delete) return updated workflow or project
func (c *client) UpdateFavorite(params sdk.FavoriteParams) (interface{}, error) {
	switch params.Type {
	case "workflow":
		var wf sdk.Workflow
		if _, err := c.PostJSON(context.Background(), "/user/favorite", params, &wf); err != nil {
			return wf, err
		}
		return wf, nil
	case "project":
		var proj sdk.Project
		if _, err := c.PostJSON(context.Background(), "/user/favorite", params, &proj); err != nil {
			return proj, err
		}
		return proj, nil
	}

	var res interface{}
	if _, err := c.PostJSON(context.Background(), "/user/favorite", params, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) UserGetSchema() (sdk.SchemaResponse, error) {
	var res sdk.SchemaResponse
	if _, err := c.GetJSON(context.Background(), "/user/schema", &res); err != nil {
		return res, err
	}
	return res, nil
}

func (c *client) ProjectFavoritesList(username string) ([]sdk.Project, error) {
	res := sdk.AuthentifiedUser{}
	mods := []RequestModifier{}
	mods = append(mods, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withFavoritesProjects", "true")
		r.URL.RawQuery = q.Encode()
	})

	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username), &res, mods...); err != nil {
		return nil, err
	}
	return res.FavoritesProjects, nil
}

func (c *client) WorkflowFavoritesList(username string) ([]sdk.Workflow, error) {
	res := sdk.AuthentifiedUser{}
	mods := []RequestModifier{}
	mods = append(mods, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withFavoritesWorkflows", "true")
		r.URL.RawQuery = q.Encode()
	})
	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username), &res, mods...); err != nil {
		return nil, err
	}
	return res.FavoritesWorkflows, nil
}
