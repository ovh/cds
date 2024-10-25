package cdsclient

import (
	"context"
	"fmt"
	"net/http"
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

func (c *client) UserGetSchemaV2(ctx context.Context, entityType string) (sdk.Schema, error) {
	res, _, code, err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v2/jsonschema/%s", entityType), nil)
	if err == nil {
		if code != http.StatusOK {
			err = fmt.Errorf("unexpected status code: %d", code)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get schema: %w", err)
	}

	return res, nil
}

func (c *client) UserUpdate(ctx context.Context, username string, u *sdk.AuthentifiedUser) error {
	if _, err := c.PutJSON(ctx, "/user/"+url.QueryEscape(username), u, u); err != nil {
		return err
	}
	return nil
}

func (c *client) UserContacts(ctx context.Context, username string) ([]sdk.UserContact, error) {
	var res []sdk.UserContact
	if _, err := c.GetJSON(ctx, "/user/"+url.QueryEscape(username)+"/contact", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) UserGpgKeyList(ctx context.Context, username string) ([]sdk.UserGPGKey, error) {
	var keys []sdk.UserGPGKey
	if _, err := c.GetJSON(ctx, fmt.Sprintf("/v2/user/%s/gpgkey", url.QueryEscape(username)), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *client) UserGpgKeyGet(ctx context.Context, keyID string) (sdk.UserGPGKey, error) {
	var key sdk.UserGPGKey
	if _, err := c.GetJSON(ctx, "/v2/user/gpgkey/"+keyID, &key); err != nil {
		return key, err
	}
	return key, nil
}

func (c *client) UserGpgKeyDelete(ctx context.Context, username string, keyID string) error {
	if _, err := c.DeleteJSON(ctx, fmt.Sprintf("/v2/user/%s/gpgkey/%s", username, keyID), nil); err != nil {
		return err
	}
	return nil
}

func (c *client) UserGpgKeyCreate(ctx context.Context, username string, publicKey string) (sdk.UserGPGKey, error) {
	key := sdk.UserGPGKey{
		PublicKey: publicKey,
	}
	if _, err := c.PostJSON(ctx, fmt.Sprintf("/v2/user/%s/gpgkey", username), key, &key); err != nil {
		return key, err
	}
	return key, nil
}
