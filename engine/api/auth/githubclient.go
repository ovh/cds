package auth

import "github.com/go-gorp/gorp"

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
