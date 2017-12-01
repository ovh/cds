package cdsclient

import (
	"fmt"
	"net/http"
	"net/url"

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

	if _, err := c.PostJSON("/login", r, &response); err != nil {
		return false, "", err
	}

	if response.Token != "" {
		return true, response.Token, nil
	}
	return true, response.Password, nil
}

func (c *client) UserList() ([]sdk.User, error) {
	res := []sdk.User{}
	if _, err := c.GetJSON("/user", &res); err != nil {
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

	code, err := c.PostJSON("/user/signup", request, nil)
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
	if _, err := c.GetJSON("/user/"+url.QueryEscape(username), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *client) UserGetGroups(username string) (map[string][]sdk.Group, error) {
	res := map[string][]sdk.Group{}
	if _, err := c.GetJSON("/user/"+url.QueryEscape(username)+"/groups", &res); err != nil {
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

	code, err := c.PostJSON("/user/"+url.QueryEscape(username)+"/reset", req, nil)
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
	if _, err := c.GetJSON("/user/"+url.QueryEscape(username)+"/confirm/"+url.QueryEscape(token), &res); err != nil {
		return false, "", err
	}
	return true, res.Password, nil
}
