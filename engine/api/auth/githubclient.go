package auth

//GithubConfig handles all config to connect to the github
type GithubConfig struct {
	ClientID     string
	ClientSecret string
}

//GithubClient is a github impl
type GithubClient struct{}

func (c *GithubClient) Init(options interface{}) error {
	return nil
}

//Authentify check username and password
func (c *GithubClient) AuthentificationURL() (string, error) {
	return "", nil
}

func (c *GithubClient) Callback(token string) error {
	return nil
}
