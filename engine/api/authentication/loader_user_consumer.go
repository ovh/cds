package authentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// LoadUserConsumerOptionFunc for auth consumer.
type LoadUserConsumerOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthUserConsumer) error

// LoadUserConsumerOptions provides all options on auth consumer loads functions.
var LoadUserConsumerOptions = struct {
	Default                          LoadUserConsumerOptionFunc
	WithAuthentifiedUser             LoadUserConsumerOptionFunc
	WithAuthentifiedUserWithContacts LoadUserConsumerOptionFunc
	WithConsumerGroups               LoadUserConsumerOptionFunc
}{
	Default:                          loadDefault,
	WithAuthentifiedUser:             loadAuthentifiedUser,
	WithAuthentifiedUserWithContacts: loadAuthentifiedUserWithContacts,
	WithConsumerGroups:               loadConsumerGroups,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthUserConsumer) error {
	return loadConsumerGroups(ctx, db, cs...)
}

func loadAuthentifiedUserWithContacts(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthUserConsumer) error {
	if err := loadAuthentifiedUser(ctx, db, cs...); err != nil {
		return err
	}

	for i := range cs {
		// Add contacts for consumer's user
		if err := user.LoadOptions.WithContacts(ctx, db, cs[i].AuthConsumerUser.AuthentifiedUser); err != nil {
			return err
		}
	}

	return nil
}

func loadAuthentifiedUser(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthUserConsumer) error {
	for _, c := range cs {
		if c.Type == sdk.ConsumerHatchery {
			return nil
		}
	}

	// Load all users for given access tokens
	users, err := user.LoadAllByIDs(ctx, db, sdk.AuthConsumersToAuthentifiedUserIDs(cs), user.LoadOptions.WithOrganization)
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
		for _, link := range mLinks[users[i].ID] {
			if grp, ok := mGroups[link.GroupID]; ok {
				users[i].Groups = append(users[i].Groups, grp)
			}
		}
	}

	mUsers := make(map[string]sdk.AuthentifiedUser)
	for i := range users {
		mUsers[users[i].ID] = users[i]
	}

	for i := range cs {
		if user, ok := mUsers[cs[i].AuthConsumerUser.AuthentifiedUserID]; ok {
			cs[i].AuthConsumerUser.AuthentifiedUser = &user
		}
	}

	return nil
}

func loadConsumerGroups(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthUserConsumer) error {
	var groupIDs []int64
	for i := range cs {
		groupIDs = append(groupIDs, cs[i].AuthConsumerUser.GroupIDs...)
	}

	// Load all groups for given ids
	gs, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}

	// Set groups on each consumers
	mGroups := gs.ToMap()
	for i := range cs {
		cs[i].AuthConsumerUser.Groups = make([]sdk.Group, 0, len(cs[i].AuthConsumerUser.GroupIDs))
		for _, groupID := range cs[i].AuthConsumerUser.GroupIDs {
			cs[i].AuthConsumerUser.Groups = append(cs[i].AuthConsumerUser.Groups, mGroups[groupID])
		}
	}

	return nil
}
