package cdsclient

import (
	"fmt"
	"net/http"

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

	code, err := c.PostJSON("/login", r, &response)
	if err != nil {
		return false, "", err
	}

	if code != http.StatusOK {
		return false, "", fmt.Errorf("Error %d", code)
	}

	if response.Token != "" {
		return true, response.Token, nil
	}
	return true, response.Password, nil
}
