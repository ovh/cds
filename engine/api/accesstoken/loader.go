package accesstoken

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadOptionFunc for access token.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AccessToken) error

// LoadOptions provides all options on access token loads functions.
var LoadOptions = struct {
	WithGroups           LoadOptionFunc
	WithAuthentifiedUser LoadOptionFunc
}{
	WithGroups:           loadGroups,
	WithAuthentifiedUser: loadAuthentifiedUser,
}

func loadGroups(ctx context.Context, db gorp.SqlExecutor, ats ...*sdk.AccessToken) error {
	// Get all access token group links for given access tokens ids
	atgs, err := getAccessTokenGroupsForAccessTokenIDs(ctx, db, sdk.AccessTokensToIDs(ats))
	if err != nil {
		return err
	}
	// Create a map of access token group links by access token ids
	mAccessTokenGroups := make(map[string][]accessTokenGroup)
	for i := range atgs {
		if _, ok := mAccessTokenGroups[atgs[i].AccessTokenID]; !ok {
			mAccessTokenGroups[atgs[i].AccessTokenID] = []accessTokenGroup{atgs[i]}
		} else {
			mAccessTokenGroups[atgs[i].AccessTokenID] = append(mAccessTokenGroups[atgs[i].AccessTokenID], atgs[i])
		}
	}

	// Get all group ids for access token group links
	groupIDs := make([]int64, len(atgs))
	for i := range atgs {
		groupIDs[i] = atgs[i].GroupID
	}
	// Get all groups and create a map of group by ids
	gs, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mGroups := make(map[int64]sdk.Group)
	for i := range gs {
		mGroups[gs[i].ID] = gs[i]
	}

	// Set groups for all given access tokens
	for i := range ats {
		atgs := mAccessTokenGroups[ats[i].ID]
		for j := range atgs {
			if g, ok := mGroups[atgs[j].GroupID]; ok {
				ats[i].Groups = append(ats[i].Groups, g)
			}
		}
	}

	return nil
}

func loadAuthentifiedUser(ctx context.Context, db gorp.SqlExecutor, ats ...*sdk.AccessToken) error {
	// Load all users for given access tokens
	users, err := user.LoadAllByIDs(ctx, db, sdk.AccessTokensToAuthentifiedUserIDs(ats), user.LoadOptions.WithDeprecatedUser)
	if err != nil {
		return err
	}

	log.Debug("loadAuthentifiedUser> users: %v", users)

	mUsers := make(map[string]sdk.AuthentifiedUser)
	for i := range users {
		mUsers[users[i].ID] = users[i]
	}

	for i := range ats {
		if user, ok := mUsers[ats[i].AuthentifiedUserID]; ok {
			ats[i].AuthentifiedUser = &user
		}
	}

	return nil
}
