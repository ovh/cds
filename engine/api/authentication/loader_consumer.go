package authentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// LoadConsumerOptionFunc for auth consumer.
type LoadConsumerOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthConsumer) error

// LoadConsumerOptions provides all options on auth consumer loads functions.
var LoadConsumerOptions = struct {
	Default              LoadConsumerOptionFunc
	WithAuthentifiedUser LoadConsumerOptionFunc
	WithConsumerGroups   LoadConsumerOptionFunc
}{
	Default:              loadDefault,
	WithAuthentifiedUser: loadAuthentifiedUser,
	WithConsumerGroups:   loadConsumerGroups,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthConsumer) error {
	return loadConsumerGroups(ctx, db, cs...)
}

func loadAuthentifiedUser(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthConsumer) error {
	// Load all users for given access tokens
	users, err := user.LoadAllByIDs(ctx, db, sdk.AuthConsumersToAuthentifiedUserIDs(cs))
	if err != nil {
		return err
	}

	// Get all links group user for user ids
	links, err := group.LoadLinksGroupUserForUserIDs(ctx, db, users.IDs())
	if err != nil {
		return err
	}
	mLinks := make(map[string][]group.LinkGroupUser)
	for i := range links {
		if _, ok := mLinks[links[i].AuthentifiedUserID]; !ok {
			mLinks[links[i].AuthentifiedUserID] = []group.LinkGroupUser{links[i]}
		} else {
			mLinks[links[i].AuthentifiedUserID] = append(mLinks[links[i].AuthentifiedUserID], links[i])
		}
	}

	// Load all groups for links
	groups, err := group.LoadAllByIDs(ctx, db, links.ToGroupIDs())
	if err != nil {
		return err
	}
	mGroups := groups.ToMap()

	// Set groups for each
	for i := range users {
		oldUserID := users[i].ID
		if _, ok := mLinks[oldUserID]; ok {
			for _, link := range mLinks[oldUserID] {
				if grp, ok := mGroups[link.GroupID]; ok {
					users[i].Groups = append(users[i].Groups, grp)
				}
			}
		}
	}

	mUsers := make(map[string]sdk.AuthentifiedUser)
	for i := range users {
		mUsers[users[i].ID] = users[i]
	}

	for i := range cs {
		if user, ok := mUsers[cs[i].AuthentifiedUserID]; ok {
			cs[i].AuthentifiedUser = &user
		}
	}

	return nil
}

func loadConsumerGroups(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthConsumer) error {
	var groupIDs []int64
	for i := range cs {
		groupIDs = append(groupIDs, cs[i].GroupIDs...)
	}

	// Load all groups for given ids
	gs, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}

	// Set groups on each consumers
	mGroups := gs.ToMap()
	for i := range cs {
		cs[i].Groups = make([]sdk.Group, 0, len(cs[i].GroupIDs))
		for _, groupID := range cs[i].GroupIDs {
			cs[i].Groups = append(cs[i].Groups, mGroups[groupID])
		}
	}

	return nil
}
