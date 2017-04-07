package auth

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/sdk"
)

//GithubConfig handles all config to connect to the github
type GithubConfig struct {
	ClientID     string
	ClientSecret string
}

//GithubCLient is a github impl
type GithubCLient struct{}

//Authentify check username and password
func (c *GithubCLient) Authentify(db gorp.SqlExecutor, username, password string) (bool, error) {
	return true, nil
}

//AuthentifyUser check password in database
func (c *GithubCLient) AuthentifyUser(db gorp.SqlExecutor, u *sdk.User, password string) (bool, error) {
	return true, nil
}

//GetCheckAuthHeaderFunc returns the func to heck http headers.
//Options is a const to switch from session to basic auth or both
func (c *GithubCLient) GetCheckAuthHeaderFunc(options interface{}) func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {

	return func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
		return nil
	}
}
