package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) UserLogin(username, password string) (bool, string, error) {
	r := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	response := struct {
		User     sdk.User `json:"user"`
		Password string   `json:"password,omitempty"`
		Token    string   `json:"token,omitempty"`
	}{}

	if _, err := c.PostJSON(context.Background(), "/login", r, &response); err != nil {
		return false, "", err
	}

	if response.Token != "" {
		return true, response.Token, nil
	}
	return true, response.Password, nil
}

func (c *client) UserList() ([]sdk.User, error) {
	res := []sdk.User{}
	if _, err := c.GetJSON(context.Background(), "/user", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) UserSignup(username, fullname, email, callback string) error {
	u := sdk.NewUser(username)
	u.Fullname = fullname
	u.Email = email

	request := sdk.UserAPIRequest{
		User:     *u,
		Callback: callback,
	}

	code, err := c.PostJSON(context.Background(), "/user/signup", request, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("Error %d", code)
	}
	return nil
}

func (c *client) UserGet(username string) (*sdk.User, error) {
	res := sdk.User{}
	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username), &res); err != nil {
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

func (c *client) UserReset(username, email, callback string) error {
	req := sdk.UserAPIRequest{
		User: sdk.User{
			Email: email,
		},
		Callback: callback,
	}

	code, err := c.PostJSON(context.Background(), "/user/"+url.QueryEscape(username)+"/reset", req, nil)
	if err != nil {
		return err
	}
	if code != http.StatusCreated {
		return fmt.Errorf("Error %d", code)
	}
	return nil
}

func (c *client) UserConfirm(username, token string) (bool, string, error) {
	res := sdk.UserAPIResponse{}
	if _, err := c.GetJSON(context.Background(), "/user/"+url.QueryEscape(username)+"/confirm/"+url.QueryEscape(token), &res); err != nil {
		return false, "", err
	}
	return true, res.Password, nil
}

// ListAllTokens Get all tokens that an user can access
func (c *client) ListAllTokens() ([]sdk.Token, error) {
	tokens := []sdk.Token{}
	if _, err := c.GetJSON(context.Background(), "/user/token", &tokens); err != nil {
		return tokens, err
	}
	return tokens, nil
}

// FindToken Get a specific token with his value to have description
func (c *client) FindToken(tokenValue string) (sdk.Token, error) {
	token := sdk.Token{}
	if code, err := c.GetJSON(context.Background(), "/user/token/"+url.QueryEscape(tokenValue), &token); err != nil {
		if code == http.StatusNotFound {
			return token, sdk.ErrTokenNotFound
		}
		return token, err
	}
	return token, nil
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

func (c *client) ProjectFavoritesList(username string) ([]sdk.Project, error) {
	res := sdk.User{}
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
	res := sdk.User{}
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

func (c *client) UserLoginCallback(ctx context.Context, request string, publicKey []byte) (sdk.AccessToken, string, error) {
	var ticker = time.NewTicker(time.Second)
	var callbackRequest = sdk.UserLoginCallbackRequest{
		RequestToken: request,
		PublicKey:    publicKey,
	}
	var accessToken sdk.AccessToken

	for {
		select {
		case <-ctx.Done():
			return accessToken, "", ctx.Err()
		case <-ticker.C:
			_, headers, _, err := c.RequestJSON(ctx, "POST", "/login/callback", callbackRequest, &accessToken)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					continue
				}
				return accessToken, "", err
			}
			jwt := headers.Get("X-CDS-JWT")
			return accessToken, jwt, nil
		}
	}
}
