package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for group.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Group) error

// LoadOptions provides all options on group loads functions.
var LoadOptions = struct {
	Default     LoadOptionFunc
	WithMembers LoadOptionFunc
}{
	Default:     loadDefault,
	WithMembers: loadMembers,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, gs ...*sdk.Group) error {
	return loadMembers(ctx, db, gs...)
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

	// Get all authentified users for migrations
	members, err := user.LoadAllByIDs(ctx, db, links.ToUserIDs())
	if err != nil {
		return err
	}
	mMembers := members.ToMapByID()

	// Set members on each groups
	for _, g := range gs {
		if _, ok := mLinks[g.ID]; ok {
			for _, link := range mLinks[g.ID] {
				if member, ok := mMembers[link.AuthentifiedUserID]; ok {
					g.Members = append(g.Members, sdk.GroupMember{
						ID:       member.ID,
						Username: member.Username,
						Fullname: member.Fullname,
						Admin:    link.Admin,
					})
				}

			}
		}
	}

	return nil
}
