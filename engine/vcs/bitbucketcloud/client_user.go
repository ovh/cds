package bitbucketcloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// User Get a single user
func (client *bitbucketcloudClient) User(ctx context.Context, username string) (User, error) {
	var user User
	url := fmt.Sprintf("/users/%s", username)
	status, body, _, err := client.get(url)
	if err != nil {
		log.Warning(ctx, "bitbucketcloudClient.User> Error %s", err)
		return user, err
	}
	if status >= 400 {
		return user, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	if err := json.Unmarshal(body, &user); err != nil {
		log.Warning(ctx, "bitbucketcloudClient.User> Unable to parse bitbucket cloud commit: %s", err)
		return user, err
	}

	return user, nil
}

// User Get a current user
func (client *bitbucketcloudClient) CurrentUser(ctx context.Context) (User, error) {
	var user User
	url := "/user"
	cacheKey := cache.Key("vcs", "bitbucketcloud", "users", client.OAuthToken, url)

	find, err := client.Cache.Get(cacheKey, &user)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
	}
	if !find {
		status, body, _, err := client.get(url)
		if err != nil {
			log.Warning(ctx, "bitbucketcloudClient.CurrentUser> Error %s", err)
			return user, sdk.WithStack(err)
		}
		if status >= 400 {
			return user, sdk.NewError(sdk.ErrUserNotFound, errorAPI(body))
		}
		if err := json.Unmarshal(body, &user); err != nil {
			return user, sdk.WithStack(err)
		}
		//Put the body on cache for 1 hour
		if err := client.Cache.SetWithTTL(cacheKey, user, 60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
		}
	}

	return user, nil
}

// User Get a current user
func (client *bitbucketcloudClient) Teams(ctx context.Context) (Teams, error) {
	var teams Teams
	url := "/teams?role=member"
	cacheKey := cache.Key("vcs", "bitbucketcloud", "users", "teams", client.OAuthToken, url)

	find, err := client.Cache.Get(cacheKey, &teams)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
	}
	if !find {
		status, body, _, err := client.get(url)
		if err != nil {
			log.Warning(ctx, "bitbucketcloudClient.Teams> Error %s", err)
			return teams, sdk.WithStack(err)
		}
		if status >= 400 {
			return teams, sdk.NewError(sdk.ErrNotFound, errorAPI(body))
		}
		if err := json.Unmarshal(body, &teams); err != nil {
			return teams, sdk.WithStack(err)
		}
		//Put the body on cache for 1 hour
		if err := client.Cache.SetWithTTL(cacheKey, teams, 60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
		}
	}

	return teams, nil
}
