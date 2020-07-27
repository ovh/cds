package github

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// User Get a single user
// https://developer.github.com/v3/users/#get-a-single-user
func (g *githubClient) User(ctx context.Context, username string) (User, error) {
	url := "/users/" + username
	status, body, _, err := g.get(ctx, url)
	if err != nil {
		log.Warning(ctx, "githubClient.User> Error %s", err)
		return User{}, err
	}
	if status >= 400 {
		return User{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	user := User{}

	//Github may return 304 status because we are using conditional request with ETag based headers
	if status == http.StatusNotModified {
		//If repo isn't updated, lets get them from cache
		k := cache.Key("vcs", "github", "users", g.OAuthToken, url)
		if _, err := g.Cache.Get(k, &user); err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
	} else {
		if err := json.Unmarshal(body, &user); err != nil {
			return User{}, err
		}
		//Put the body on cache for one hour and one minute
		k := cache.Key("vcs", "github", "users", g.OAuthToken, url)
		if err := g.Cache.SetWithTTL(k, user, 61*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
	}

	return user, nil
}
