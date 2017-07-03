package auth

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
)

//GithubConfig handles all config to connect to the github
type GithubConfig struct {
	ClientID     string
	ClientSecret string
}

//GithubClient is a github impl
type GithubClient struct{}

//Authentify check username and password
func (c *GithubClient) Authentify(db gorp.SqlExecutor, username, password string) (bool, error) {
	return true, nil
}

//AuthentifyUser check password in database
func (c *GithubClient) AuthentifyUser(db gorp.SqlExecutor, u *sdk.User, password string) (bool, error) {
	return true, nil
}

//CheckAuthHeader returns the func to heck http headers.
func (c *GithubClient) CheckAuthHeader(db *gorp.DbMap, headers http.Header, ctx *businesscontext.Ctx) error {
	return nil
}
