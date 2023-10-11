package hooks

import (
	"context"
	"fmt"
	"github.com/rockbears/log"
	"regexp"
	"strings"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

/**
 * Create Repository: hooks:repository:bitbucketserver-my_bibucket_server-my/repo
 * Receive Event:
 *    1. Save event:  -> hooks:events:repository:bitbucketserver-my_bibucket_server-my/repo
 *    2. Insert event_key in inprogress list: -> hooks:queue:repository:event:inprogress
 *    3. Enqueue event_key for scheduling: ->  hooks:queue:repository:event
 *
 */

var (
	repositoryLock               = cache.Key("hooks", "lock", "repository")
	repositoryRootKey            = cache.Key("hooks", "repository")
	repositoryEventRootKey       = cache.Key("hooks", "events", "repository")
	repositoryEventInProgressKey = cache.Key(repositoryEventQueue, "inprogress")
	repositoryEventQueue         = cache.Key("hooks", "queue", "repository", "event")
	repositoryEventCallbackQueue = cache.Key("hooks", "queue", "repository", "event", "callback")
	repositoryEventLockRootKey   = cache.Key("hooks", "events", "lock")
)

func (d *dao) GetRepositoryMemberKey(vcsName, repoName string) string {
	return fmt.Sprintf("%s-%s", vcsName, repoName)
}

func (d *dao) FindRepository(ctx context.Context, repoKey string) *sdk.HookRepository {
	key := cache.Key(repositoryRootKey, repoKey)
	hr := &sdk.HookRepository{}
	find, err := d.store.Get(key, hr)
	if err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "cannot get from cache %s", key))
	}
	if find {
		return hr
	}
	return nil
}

func (d *dao) CreateRepository(ctx context.Context, vcsServerType, vcsServerName, repoName string) (*sdk.HookRepository, error) {
	// Create a task for the current repository
	repoKey := d.GetRepositoryMemberKey(vcsServerName, repoName)
	log.Info(ctx, "creating repository %s", repoKey)
	hr := &sdk.HookRepository{
		RepositoryName: repoName,
		VCSServerName:  vcsServerName,
		VCSServerType:  vcsServerType,
	}
	if err := d.store.SetAdd(repositoryRootKey, repoKey, hr); err != nil {
		return nil, err
	}
	return hr, nil
}

func (d *dao) DeleteRepository(ctx context.Context, vcsserver, repo string) error {
	key := cache.Key(repositoryRootKey, d.GetRepositoryMemberKey(vcsserver, repo))
	return d.store.Delete(key)
}

func (d *dao) ListRepositories(ctx context.Context, filter string) ([]string, error) {
	var filteredRepos []string
	repos, err := d.store.Keys(cache.Key(repositoryRootKey, "*"))
	if err != nil {
		return nil, err
	}
	log.Warn(ctx, "%s", filter)
	if filter == "" {
		for _, r := range repos {
			filteredRepos = append(filteredRepos, strings.TrimPrefix(r, repositoryRootKey+":"))
		}
		return filteredRepos, nil
	}

	reg, err := regexp.Compile(filter)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	for _, r := range repos {
		r = strings.TrimPrefix(r, repositoryRootKey+":")
		log.Info(ctx, "%s", r)
		if reg.MatchString(r) {
			filteredRepos = append(filteredRepos, r)
		}
	}

	return filteredRepos, nil
}
