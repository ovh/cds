package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadOptionFunc for group.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Group) error

// LoadOptions provides all options on group loads functions.
var LoadOptions = struct {
	WithMembers LoadOptionFunc
}{
	WithMembers: loadMembers,
}

func loadMembers(ctx context.Context, db gorp.SqlExecutor, gs ...*sdk.Group) error {
	groupIDs := sdk.GroupPointersToIDs(gs)

	// Get all links group user for groupd ids
	links, err := LoadLinksGroupUserForGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mLinks := make(map[int64][]LinkGroupUser)
	for i := range links {
		if _, ok := mLinks[links[i].GroupID]; !ok {
			mLinks[links[i].GroupID] = []LinkGroupUser{links[i]}
		} else {
			mLinks[links[i].GroupID] = append(mLinks[links[i].GroupID], links[i])
		}
	}

	log.Debug("group.loadMembers> links: %v", links)

	// Get all users for links
	members, err := user.LoadDeprecatedUsersWithoutAuthByIDs(ctx, db, links.ToUserIDs())
	if err != nil {
		return err
	}
	mMembers := make(map[int64]sdk.User, len(members))
	for i := range members {
		mMembers[members[i].ID] = members[i]
	}

	log.Debug("group.loadMembers> members: %v", members)

	// Set links on each groups
	for _, g := range gs {
		if _, ok := mLinks[g.ID]; ok {
			for _, link := range mLinks[g.ID] {
				if member, ok := mMembers[link.UserID]; ok {
					member.GroupAdmin = link.Admin
					g.Members = append(g.Members, member)
				}
			}
		}
	}

	return nil
}
