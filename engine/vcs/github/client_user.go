package github

import (
	"encoding/json"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// User Get a single user
// https://developer.github.com/v3/users/#get-a-single-user
func (g *githubClient) User(username string) (User, error) {
	url := "/users/" + username
	status, body, _, err := g.get(url)
	if err != nil {
		log.Warning("githubClient.User> Error %s", err)
		return User{}, err
	}
	if status >= 400 {
		return User{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	user := User{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		g.Cache.Get(cache.Key("vcs", "github", "users", g.OAuthToken, url), &user)
	} else {
		if err := json.Unmarshal(body, &user); err != nil {
			return User{}, err
		}
		//Put the body on cache for one hour and one minute
		g.Cache.SetWithTTL(cache.Key("vcs", "github", "users", g.OAuthToken, url), user, 61*60)
	}

	return user, nil
}
