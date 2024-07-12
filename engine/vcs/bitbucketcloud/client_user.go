package bitbucketcloud

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// User Get a single user
func (client *bitbucketcloudClient) User(ctx context.Context, username string) (User, error) {
	var user User
	url := fmt.Sprintf("/users/%s", username)
	status, body, _, err := client.get(ctx, url)
	if err != nil {
		log.Warn(ctx, "bitbucketcloudClient.User> Error %s", err)
		return user, err
	}
	if status >= 400 {
		return user, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	if err := sdk.JSONUnmarshal(body, &user); err != nil {
		log.Warn(ctx, "bitbucketcloudClient.User> Unable to parse bitbucket cloud commit: %s", err)
		return user, err
	}

	return user, nil
}

// User Get a current user
func (client *bitbucketcloudClient) CurrentUser(ctx context.Context) (User, error) {
	var user User
	url := "/user"
	cacheKey := cache.Key("vcs", "bitbucketcloud", "users", sdk.Hash512(client.username), url)

	find, err := client.Cache.Get(ctx, cacheKey, &user)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
	}
	if !find {
		status, body, _, err := client.get(ctx, url)
		if err != nil {
			log.Warn(ctx, "bitbucketcloudClient.CurrentUser> Error %s", err)
			return user, sdk.WithStack(err)
		}
		if status >= 400 {
			return user, sdk.NewError(sdk.ErrUserNotFound, errorAPI(body))
		}
		if err := sdk.JSONUnmarshal(body, &user); err != nil {
			return user, sdk.WithStack(err)
		}
		//Put the body on cache for 1 hour
		if err := client.Cache.SetWithTTL(ctx, cacheKey, user, 60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
		}
	}

	return user, nil
}

// Workspaces Returns a list of workspaces accessible by the authenticated user.
func (client *bitbucketcloudClient) Workspaces(ctx context.Context) (Workspaces, error) {
	var workspaces Workspaces
	url := "/workspaces?role=member"

	cacheKey := cache.Key("vcs", "bitbucketcloud", "users", "workspaces", sdk.Hash512(client.username), url)

	find, err := client.Cache.Get(ctx, cacheKey, &workspaces)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
	}
	if !find {
		status, body, _, err := client.get(ctx, url)
		if err != nil {
			log.Warn(ctx, "bitbucketcloudClient.Teams> Error %s username:%v status:%d body:%v", err, client.username, status, string(body))
			return workspaces, sdk.WithStack(err)
		}
		if status >= 400 {
			return workspaces, sdk.NewError(sdk.ErrNotFound, errorAPI(body))
		}
		if err := sdk.JSONUnmarshal(body, &workspaces); err != nil {
			return workspaces, sdk.WithStack(err)
		}
		// Put the body on cache for 1 hour
		if err := client.Cache.SetWithTTL(ctx, cacheKey, workspaces, 60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
		}
	}

	return workspaces, nil
}
