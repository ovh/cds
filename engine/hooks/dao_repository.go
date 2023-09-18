package hooks

import (
	"context"
	"fmt"
	"github.com/rockbears/log"

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
	repositoryRootKey            = cache.Key("hooks", "repository")
	repositoryEventRootKey       = cache.Key("hooks", "events", "repository")
	repositoryEventInProgressKey = cache.Key(repositoryEventQueue, "inprogress")
	repositoryEventQueue         = cache.Key("hooks", "queue", "repository", "event")
	repositoryEventCallbackQueue = cache.Key("hooks", "queue", "repository", "event", "callback")
	repositoryEventLockRootKey   = cache.Key("hooks", "events", "lock")
)

func (d *dao) GetRepositoryMemberKey(vcsType, vcsName, repoName string) string {
	return fmt.Sprintf("%s-%s-%s", vcsType, vcsName, repoName)
}

func (d *dao) FindRepository(ctx context.Context, repoKey string) *sdk.HookRepository {
	key := cache.Key(repositoryRootKey, repoKey)
	hr := &sdk.HookRepository{}
	find, err := d.store.Get(key, hr)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", key, err)
	}
	if find {
		return hr
	}
	return nil
}

func (d *dao) CreateRepository(ctx context.Context, repoKey, vcsServerType, vcsServerName, repoName string) (*sdk.HookRepository, error) {
	// Create a task for the current repository
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
